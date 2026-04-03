package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	serialConfig serial.Config
	mu           sync.Mutex
}
type CSIFrame struct {
	Rssi      int       `json:"rssi"`
	Index     int       `json:"index"`
	Len       int       `json:"len"`
	Raw       []int     `json:"raw"`       // 256个原始整数
	Amplitude []float64 `json:"amplitude"` // 幅度
	Phase     []float64 `json:"phase"`     // 相位
}

var csiRegex = regexp.MustCompile(`rssi:(-?\d+)\s+index:(\d+)\s+len:(\d+)\s+data:\[(.*?)\]`)

var config = &serial.Config{
	Name:        "COM4",
	Baud:        115200,
	ReadTimeout: time.Second * 1,
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.serialConfig = *config // 初始化默认配置
	a.Reconnect()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("你好 Hello %s, It's show time!", name)
}

// Reconnect 串口连接
func (a *App) Reconnect() {
	a.mu.Lock()
	// 1. 如果已有协程在运行，先停止它
	if a.cancelFunc != nil {
		a.cancelFunc()
		time.Sleep(2 * time.Second) // 给一点时间让旧串口完成释放
	}

	// 2. 创建新的 Context
	subCtx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel
	a.mu.Unlock()
	// 3. 启动新协程
	go a.readSerial(subCtx)
}

func (a *App) readSerial(ctx context.Context) {
	a.mu.Lock()
	config := a.serialConfig
	a.mu.Unlock()
	port, err := serial.OpenPort(&config)
	if err != nil {
		runtime.LogErrorf(a.ctx, "串口[%s]打开失败: %v", config.Name, err)
		runtime.EventsEmit(a.ctx, "connection-status", "["+config.Name+"]失败: "+err.Error())
		return
	}
	runtime.LogInfo(a.ctx, "串口["+config.Name+"]打开成功")
	runtime.EventsEmit(a.ctx, "connection-status", "连接成功")
	defer func() {
		port.Close()
		runtime.LogInfo(a.ctx, "串口已关闭，协程退出")
	}()
	reader := bufio.NewReader(port)
	for {
		select {
		case <-ctx.Done(): // 关键：收到取消信号，退出循环
			return
		default:
			line, err := reader.ReadString('\n') // 读取到换行符
			if err != nil {
				runtime.LogErrorf(a.ctx, "读取错误: %v", err)
				continue
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			frame, err := parseCSI(line)
			if err != nil {
				runtime.LogWarningf(a.ctx, "解析失败: %v | 原始: %s", err, line[:min(len(line), 50)])
				continue
			}
			runtime.LogInfof(a.ctx, "解析成功: RSSI=%d, Index=%d, 子载波=%d个",
				frame.Rssi, frame.Index, len(frame.Amplitude))
			runtime.EventsEmit(a.ctx, "csi-data", frame)
		}
	}
}

func parseCSI(line string) (*CSIFrame, error) {
	matches := csiRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("格式不匹配")
	}
	rssi, _ := strconv.Atoi(matches[1])
	index, _ := strconv.Atoi(matches[2])
	length, _ := strconv.Atoi(matches[3])
	dataStr := matches[4]
	raw := []int{}
	for _, s := range strings.Split(dataStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		val, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		raw = append(raw, val)
	}
	frame := &CSIFrame{
		Rssi:  rssi,
		Index: index,
		Len:   length,
		Raw:   raw,
	}
	// 计算幅度和相位 (实部, 虚部交替)
	for i := 0; i < len(raw)-1; i += 2 {
		real := float64(raw[i])
		imag := float64(raw[i+1])

		amp := math.Sqrt(real*real + imag*imag)
		phase := math.Atan2(imag, real) * 180 / math.Pi // 转角度

		frame.Amplitude = append(frame.Amplitude, amp)
		frame.Phase = append(frame.Phase, phase)
	}
	return frame, nil
}

// UpdateSerialConfig 更新串口配置
// 前端可以传入参数，如：UpdateSerialConfig("COM5", 921600)
func (a *App) UpdateSerialConfig(name string, baud int) {
	a.mu.Lock()
	a.serialConfig.Name = name
	a.serialConfig.Baud = baud
	a.mu.Unlock()

	runtime.LogInfof(a.ctx, "配置已更新: %s, %d. 正在尝试重连...", name, baud)

	// 更新完配置后，通常需要自动重连才能生效
	a.Reconnect()
}
