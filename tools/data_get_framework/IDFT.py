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


# ================== Parser ==================
class IFrontParser(ABC):
    @abstractmethod
    def parse(self, line: str) -> list[int]:
        pass


class DefaultFrontParser(IFrontParser):
    def parse(self, line: str) -> list[int]:
        match = re.search(r'data:\s*\[([^\]]+)\]', line)
        if not match:
            return None

        data_str = match.group(1)
        data_list = []
        for x in data_str.split(','):
            try:
                data_list.append(int(x.strip()))
            except:
                continue

        return data_list if len(data_list) > 0 else None


# ================== 串口线程 ==================
def serial_thread_func(port, baud, q, parser):
    with serial.Serial(port, baud, timeout=1) as ser:
        print(f"[串口] 打开 {port} @ {baud}bps")

        while True:
            try:
                line = ser.readline().decode(errors='ignore')

                if not line:
                    continue

                data = parser.parse(line)
                if data is None:
                    continue

                if not q.full():
                    q.put(data)

            except Exception as e:
                print("[串口] 错误:", e)


# ================== 信号处理 ==================
def complex_conjugate(c):
    return complex(c.real, -c.imag)


def csi_ratio(a, b):
    conj_b = complex_conjugate(b)
    product = a * conj_b
    power_b = b.real**2 + b.imag**2

    if power_b < 1e-10:
        return complex(0, 0)

    return complex(product.real / power_b, product.imag / power_b)


# ================== 主窗口 ==================
class MainWindow(QtWidgets.QMainWindow):
    def __init__(self, data_queue):
        super().__init__()

        self.setWindowTitle("CSI 频谱分析")
        self.resize(1000, 600)

        self.data_queue = data_queue

        # ===== 时域缓存 =====
        self.fft_size = 128
        self.buffer = deque(maxlen=self.fft_size)

        # ===== UI =====
        self.plot_widget = pg.PlotWidget()
        self.setCentralWidget(self.plot_widget)

        self.plot_widget.setTitle("Frequency Spectrum")
        self.plot_widget.setLabel('left', 'Amplitude')
        self.plot_widget.setLabel('bottom', 'Frequency Bin')

        self.curve = self.plot_widget.plot(pen='c')

        # ===== Timer =====
        self.timer = QtCore.QTimer()
        self.timer.timeout.connect(self.update_plot)
        self.timer.start(30)

    # ===== CSI解析 =====
    def data_parse(self, csi_frame):
        try:
            channel = 9

            c1 = complex(csi_frame[channel*2+1], csi_frame[channel*2])
            c2 = complex(csi_frame[channel*2+3], csi_frame[channel*2+2])

            ratio = csi_ratio(c1, c2)

            return abs(ratio)

        except:
            return None

    # ===== FFT =====
    def compute_fft(self):
        if len(self.buffer) < self.fft_size:
            return None, None

        data = np.array(self.buffer)

        # 去均值（非常关键）
        data = data - np.mean(data)

        # 加窗（Hanning）
        window = np.hanning(len(data))
        data = data * window

        fft = np.fft.fft(data)
        fft = np.abs(fft)

        # 只取一半
        fft = fft[:len(fft)//2]

        return np.arange(len(fft)), fft

    # ===== 更新 =====
    def update_plot(self):
        # 吃掉队列
        while not self.data_queue.empty():
            frame = self.data_queue.get()
            val = self.data_parse(frame)

            if val is not None:
                self.buffer.append(val)

        # 计算FFT
        x, y = self.compute_fft()

        if x is not None:
            self.curve.setData(x, y)


# ================== 参数 ==================
def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", default="COM3")
    parser.add_argument("--baud", type=int, default=921600)
    return parser.parse_args()


# ================== 主程序 ==================
if __name__ == "__main__":
    q = queue.Queue(maxsize=2000)
    parser = DefaultFrontParser()
    args = parse_args()

    t = threading.Thread(
        target=serial_thread_func,
        args=(args.port, args.baud, q, parser),
        daemon=True
    )
    t.start()

    app = QtWidgets.QApplication(sys.argv)
    win = MainWindow(q)
    win.show()
    sys.exit(app.exec())