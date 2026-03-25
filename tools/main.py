import serial
import numpy as np
import re

# =============================
# 串口配置
# =============================
SERIAL_PORT = "COM3"
BAUDRATE = 115200

# =============================
# 提取 {...} 数据
# =============================
pattern = re.compile(r"\{([^\}]*)\}")

def parse_csi_line(line):
    """
    从一行日志中提取 CSI 数据数组
    """
    match = pattern.search(line)
    if not match:
        return None

    raw = match.group(1)

    try:
        data = [int(x.strip()) for x in raw.split(",") if x.strip() != ""]
        return data
    except:
        return None


def compute_amplitude(csi_raw):
    """
    I/Q → 幅度
    """
    if len(csi_raw) % 2 != 0:
        return None

    iq = np.array(csi_raw, dtype=np.float32)
    I = iq[0::2]
    Q = iq[1::2]

    amplitude = np.sqrt(I**2 + Q**2)

    return amplitude


def remove_zero_padding(amplitude):
    """
    去掉前后全0区域（非常关键）
    """
    # 找非零区域
    nonzero_idx = np.where(amplitude > 1e-6)[0]

    if len(nonzero_idx) == 0:
        return None

    start = nonzero_idx[0]
    end = nonzero_idx[-1]

    return amplitude[start:end+1]


# =============================
# 主循环
# =============================
def main():
    ser = serial.Serial(SERIAL_PORT, BAUDRATE, timeout=1)

    print(f"Listening on {SERIAL_PORT}...")

    while True:
        try:
            line = ser.readline().decode(errors="ignore")
            print("Raw line:", line.strip())
            
            if "data:" not in line:
                continue

            csi_raw = parse_csi_line(line)
            if csi_raw is None:
                continue

            amplitude = compute_amplitude(csi_raw)
            if amplitude is None:
                continue

            amplitude = remove_zero_padding(amplitude)
            if amplitude is None:
                continue

            # =============================
            # 打印结果
            # =============================
            print("Amplitude (len={}):".format(len(amplitude)))
            print(np.round(amplitude, 2))
            print("-" * 60)

        except KeyboardInterrupt:
            print("Exit.")
            break
        except Exception as e:
            print("Error:", e)


if __name__ == "__main__":
    main()