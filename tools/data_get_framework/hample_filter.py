##vibe模板 直接把这个数据接收的模板丢给ai，让ai在这个基础上写代码
# ## 数据可视化库请选用兼容pyqt6的
import argparse
import math
import sys
import serial
import threading
import queue
import numpy as np
from abc import ABC, abstractmethod
import re
from collections import deque

import pyqtgraph as pg
from pyqtgraph.Qt import QtWidgets, QtCore
from typing import Protocol 

class IFrontParser(ABC):
    @abstractmethod
    def parse(self, line:str) -> list[int]:
        pass

class DefaultFrontParser(IFrontParser):
    def parse(self, line:str) -> list[int]:
        match = re.search(r'data:\s*\[([^\]]+)\]', line)
        if not match:
            return None
        data_str = match.group(1)
        data_list = []
        for x in data_str.split(','):
            x = x.strip()
            try:
                data_list.append(int(x))
            except ValueError:
                continue
        
        return data_list


def serial_thread_func(
    port: str, 
    baud: int, 
    q: queue.Queue, 
    front_parser: IFrontParser,
   
):
    with serial.Serial(port, baud, timeout=1) as ser:
        print(f"[串口] 打开 {port} @ {baud}bps")
        
        while True:
            try:
                line = ser.readline().decode('utf-8')
                if not line:
                    continue
                
                csi_frame: list[int] = front_parser.parse(line)
                if csi_frame is None:
                    continue
                
                q.put(csi_frame)
                
            except Exception as e:
                print("[串口] 错误:", e)
                continue

class ISlideWindow(Protocol):
    def get_process_value(self) -> float:
        pass
    def put(self, item: float) -> None:
        pass 

class SimpleWindow(ISlideWindow):
    def __init__(self, size: int, session: object):
        self.session = session
        self.size = size
        self.items = queue.Queue(maxsize=size)
    
    def get_average(self) -> float:
        if self.items.empty():
            return 0.0
        return sum(self.items.queue) / self.items.qsize()
    
    def get_process_value(self):
        return self.get_average()
    
    def put(self, item: float) -> None:
        if self.items.full():
            self.items.get()
            self.items.put(item)
        else:
            self.items.put(item)

class HampleWindow:
    def __init__(self, size: int, session: object):
        self.session = session
        self.size = size
        if size % 2 == 0:
            size += 1  # 确保窗口大小为奇数
        
        self.items = queue.Queue(maxsize=size)
    
    def get_process_value(self) -> float:
        if not self.items.full():
            return 0.0  # 窗口未填满时返回0
        else:
            np_items = np.array(list(self.items.queue))
            median_num = np.median(np_items)  # 计算中位数
            MAD = 1.4826 * np.median(np.abs(np_items - median_num))  # 计算绝对中位差
            for index in range(len(np_items)):
                if np.abs(np_items[index] - median_num) > 3 * MAD:
                    np_items[index] = median_num  # 将异常值替换为中位数
            return np.mean(np_items)  # 返回窗口均值

    def put(self, item: float) -> None:
        if self.items.full():
            self.items.get()
            self.items.put(item)
        else:
            self.items.put(item)


class MainWindow(QtWidgets.QMainWindow):
    def __init__(self, data_queue: queue.Queue):
        self.index = 0
        super().__init__()
        self.max_points = 100
        # 原始数据
        self.raw_x_data = deque(maxlen=self.max_points)
        self.raw_y_data = deque(maxlen=self.max_points)
        # 滤波后数据
        self.filtered_x_data = deque(maxlen=self.max_points)
        self.filtered_y_data = deque(maxlen=self.max_points)
        
        self.data_queue = data_queue
        self.setWindowTitle("WiFi CSI - 原始数据 vs Hample滤波")
        self.resize(800, 600)
        
        # 创建绘图区域
        self.plot_widget = pg.PlotWidget()
        self.setCentralWidget(self.plot_widget)
        self.plot_widget.setTitle("CSI Amplitude - 原始数据(蓝色) vs Hample滤波(红色)")
        self.plot_widget.setLabel('left', 'Amplitude')
        self.plot_widget.setLabel('bottom', 'Time (samples)')
        self.plot_widget.addLegend()
        
        # 原始数据曲线（蓝色）
        self.raw_curve = self.plot_widget.plot(pen=pg.mkPen('b', width=2), name='Raw Data')
        # 滤波后数据曲线（红色）
        self.filtered_curve = self.plot_widget.plot(pen=pg.mkPen('r', width=2), name='Hample Filtered')
        
        # 定时器
        self.timer = QtCore.QTimer()
        self.timer.timeout.connect(self.update_plot)
        self.timer.start(30)  # 约 30 FPS
        
        # 创建Hample滤波窗口
        self.slide_window = HampleWindow(size=12, session=self)
        
        # 存储最近处理的数据用于滤波
        self.recent_raw_value = 0.0
        
    def data_parse(self, csi_frame: list) -> tuple:
        """
        解析CSI数据，返回原始值和滤波后的值
        csi_frame 的数据格式是[I0, Q0, I1, Q1, ...] 
        先存储实部(I)，再存储虚部(Q)
        """
        # 转换为复数并计算幅度
        csi_complex = np.array([
            complex(csi_frame[i], csi_frame[i+1])  # I + jQ
            for i in range(0, len(csi_frame), 2)
        ])
        csi_amp = np.abs(csi_complex)
        
        # 使用第10个子载波（索引9）
        raw_value = csi_amp[21]
        
        # 进行Hample滤波
        self.slide_window.put(raw_value)
        filtered_value = self.slide_window.get_process_value()
        
        return raw_value, filtered_value

    def update_plot(self):
        # 一次性吃完队列，避免堆积
        while not self.data_queue.empty():
            csi_frame = self.data_queue.get()
            raw_value, filtered_value = self.data_parse(csi_frame)
            
            self.index += 1
            
            # 存储原始数据
            self.raw_x_data.append(self.index)
            self.raw_y_data.append(raw_value)
            
            # 存储滤波后数据（只有当滤波窗口填满后才显示）
            if filtered_value != 0.0:  # 窗口填满后才添加
                self.filtered_x_data.append(self.index)
                self.filtered_y_data.append(filtered_value)
        
        # 更新原始数据曲线
        if len(self.raw_x_data) > 0:
            t0 = self.raw_x_data[0]
            x = [i - t0 for i in self.raw_x_data]
            self.raw_curve.setData(x, list(self.raw_y_data))
        
        # 更新滤波后数据曲线
        if len(self.filtered_x_data) > 0:
            t0 = self.filtered_x_data[0] if len(self.filtered_x_data) > 0 else 0
            x = [i - t0 for i in self.filtered_x_data]
            self.filtered_curve.setData(x, list(self.filtered_y_data))
        
        # 可选：自动调整Y轴范围
        if len(self.raw_y_data) > 0:
            max_y = max(max(self.raw_y_data), max(self.filtered_y_data) if self.filtered_y_data else 0)
            min_y = min(min(self.raw_y_data), min(self.filtered_y_data) if self.filtered_y_data else 0)
            if max_y > min_y:
                self.plot_widget.setYRange(min_y - 5, max_y + 5)

def parse_args():
    parser = argparse.ArgumentParser(description="WiFi CSI 人体运动检测可视化")
    parser.add_argument("--port", default="COM3")
    parser.add_argument("--baud", type=int, default=921600)
    parser.add_argument("--max-channels", type=int, default=3)
    parser.add_argument("--max-history", type=int, default=300)
    return parser.parse_args()

if __name__ == "__main__":
    queue_ = queue.Queue(maxsize=1000)
    parser = DefaultFrontParser()
    args = parse_args()
    thread = threading.Thread(
        target=serial_thread_func,
        args=(args.port, args.baud, queue_, parser),
        daemon=True
    )
    thread.start()
    app = QtWidgets.QApplication(sys.argv)
    win = MainWindow(queue_)
    win.show()
    sys.exit(app.exec())