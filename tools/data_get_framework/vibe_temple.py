
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
        #print(f"[算法] 运动检测模式：人动 → 波形跳动")
        
        while True:
            try:
                line = ser.readline().decode('utf-8')
                #print(line)
                if not line:
                
                    continue
                
                csi_frame: list[int] = front_parser.parse(line)
                if csi_frame is None:
                    
                    continue
                
                
                q.put(csi_frame)
                
                
            except Exception as e:
                
                print("[串口] 错误:", e)
                continue

class SimpleWindow:
    def __init__(self, size: int, session: object):
        self.session = session
        self.size = size
        self.items = queue.Queue(maxsize=size)
    def get_average(self) -> float:
        #return self.items.get()
        if self.items.empty():
            return 0.0
        return sum(self.items.queue) / self.items.qsize()
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
        self.max_points = 300
        self.x_data = deque(maxlen=self.max_points)
        self.y_data = deque(maxlen=self.max_points)
        self.data_queue = data_queue
        self.setWindowTitle("WiFi CSI")
        self.resize(800, 600)
        self.plot_widget = pg.PlotWidget()
        self.setCentralWidget(self.plot_widget)
        self.plot_widget.setTitle("Data View")
        self.plot_widget.setLabel('left', 'fs')
        self.plot_widget.setLabel('bottom', 'Time')
        self.curve = self.plot_widget.plot(pen='y')
        self.timer = QtCore.QTimer()
        self.timer.timeout.connect(self.update_plot)
        self.timer.start(15)  # 约 30 FPS
        16
        self.slide_window = SimpleWindow(size=16,session=self)
        
    def data_parse(self, csi_frame: list) -> float:
        

        
        return 0.0

    def update_plot(self):
        # 一次性吃完队列，避免堆积
        while not self.data_queue.empty():
            csi_frame = self.data_queue.get()
            ret = self.data_parse(csi_frame)

            self.index += 1

            self.x_data.append(self.index)
            self.y_data.append(ret)
            


        if len(self.x_data) > 0:
            # 转换为相对时间（更直观）
            t0 = self.x_data[0]
            x = [i - t0 for i in self.x_data]

            self.curve.setData(x, self.y_data)

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
