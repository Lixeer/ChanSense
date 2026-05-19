package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
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

	// 数据缓存
	csiCache     []*CSIFrame // 缓存切片
	head         int         // 头指针
	size         int         // 当前缓存数量
	maxCacheSize int         // 最大缓存数量：2000
	index        int
}
type CSIFrame struct {
	Index     int       `json:"index"`
	Raw       []int16   `json:"raw"`       // 256个原始整数
	Amplitude []float64 `json:"amplitude"` // 幅度
	Phase     []float64 `json:"phase"`     // 相位
}

var csiRegex = regexp.MustCompile(`index:(\d+).*?data:\[(.*?)\]`)
var csiRegexData = regexp.MustCompile(`data:\[(.*?)\]`)

var config = &serial.Config{
	Name:        "COM35",
	Baud:        115200,
	ReadTimeout: time.Second * 1,
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		maxCacheSize: 2000, // 初始化最大缓存限制
		head:         0,
		index:        0,
		csiCache:     make([]*CSIFrame, 2000),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.serialConfig = *config // 初始化默认配置
	a.Reconnect()
}

// Reconnect 串口连接
func (a *App) Reconnect() {
	runtime.EventsEmit(a.ctx, "connection-status", "等待串口停止")
	a.mu.Lock()
	// 等待串口释放
	if a.cancelFunc != nil {
		a.cancelFunc()
		time.Sleep(1 * time.Second)
	}

	// 创建新的 Context
	subCtx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel
	a.mu.Unlock()
	// 启动新协程
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
			a.index++
			frame.Index = a.index
			if err != nil {
				runtime.LogWarningf(a.ctx, "解析失败: %v | 原始: %s", err, line[:min(len(line), 50)])
				continue
			}
			a.mu.Lock()
			a.head = (a.head + 1) % a.maxCacheSize
			a.csiCache[a.head] = frame
			if a.size < a.maxCacheSize {
				a.size++
			}
			a.mu.Unlock()
			// runtime.LogInfof(a.ctx, "解析成功: Index=%d, 子载波=%d个", frame.Index, len(frame.Amplitude))
			runtime.EventsEmit(a.ctx, "csi-data", frame)
		}
	}
}

func parseCSI(line string) (*CSIFrame, error) {
	matches := csiRegexData.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("格式不匹配")
	}
	// index, _ := strconv.Atoi(matches[1])
	dataStr := matches[1]
	count := strings.Count(dataStr, ",") + 1
	raw := make([]int16, 0, count)
	for _, s := range strings.Split(dataStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		val, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		raw = append(raw, int16(val))
	}
	frame := &CSIFrame{
		Index: 0, // index will be set to 0 since we're not parsing it
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

// 手动保存
// func (a *App) SaveTextToFile(content string, defaultFilename string) error {
// 	// 弹出保存文件对话框
// 	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
// 		Title: "保存文本文件",
// 		Filters: []runtime.FileFilter{
// 			{
// 				DisplayName: "Text Files",
// 				Pattern:     "*.txt",
// 			},
// 		},
// 		DefaultFilename: defaultFilename, // 使用变量设置默认文件名
// 	})
// 	if err != nil || savePath == "" {
// 		return err
// 	}

// 	// 写入文件
// 	return os.WriteFile(savePath, []byte(content), 0644)
// }

func (a *App) AutoSaveTextToFile(saveIndex int, history int, filename string) error {
	// 获取当前运行的 exe 文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}
	// 获取 exe 所在的目录
	exeDir := filepath.Dir(exePath)

	saveAllDir := filepath.Join(exeDir, "CSIFrame")
	if err := os.MkdirAll(saveAllDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建存档文件夹失败: %w", err)
	}
	finalAllPath := filepath.Join(saveAllDir, filename)

	saveDataDir := filepath.Join(exeDir, "Data")
	if err := os.MkdirAll(saveDataDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建数据文件夹失败: %w", err)
	}
	finalDataPath := filepath.Join(saveDataDir, filename)

	frames := a.GetCachedCSI(saveIndex, history)
	justRaws := make([][]int16, len(frames))
	for i, frame := range frames {
		if frame != nil {
			justRaws[i] = frame.Raw
		}
	}

	jsonAllData, err := json.Marshal(frames)
	if err != nil {
		return fmt.Errorf("序列化全量数据失败: %w", err)
	}
	err = os.WriteFile(finalAllPath, jsonAllData, 0644)
	if err != nil {
		return fmt.Errorf("写入全量文件失败: %w", err)
	}

	jsonRawData, err := json.Marshal(justRaws)
	if err != nil {
		return fmt.Errorf("序列化Raw数据失败: %w", err)
	}
	err = os.WriteFile(finalDataPath, jsonRawData, 0644)
	if err != nil {
		return fmt.Errorf("写入Raw数据文件失败: %w", err)
	}
	return nil
}

func (a *App) GetCachedCSI(saveIndex int, count int) []*CSIFrame {
	a.mu.Lock()
	defer a.mu.Unlock()
	diff := max(a.index-saveIndex, 0)
	if count+diff > a.size {
		count = a.size - diff
	}
	if count <= 0 {
		return []*CSIFrame{}
	}
	result := make([]*CSIFrame, 0, count)
	// 头指针减去差值和数量，得到起始位置
	start := (a.head - diff - count + 1 + a.maxCacheSize) % a.maxCacheSize
	for i := 0; i < count; i++ {
		idx := (start + i) % a.maxCacheSize
		frame := a.csiCache[idx]
		if frame != nil {
			result = append(result, frame)
		}
	}
	return result
}

func (a *App) ClearSavedData() {
	exePath, err := os.Executable()
	if err != nil {
		runtime.LogErrorf(a.ctx, "获取程序路径失败: %v", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	saveAllDir := filepath.Join(exeDir, "CSIFrame")
	saveDataDir := filepath.Join(exeDir, "Data")

	err = os.RemoveAll(saveAllDir)
	if err != nil {
		runtime.LogErrorf(a.ctx, "清空存档文件夹失败: %v", err)
	} else {
		runtime.LogInfo(a.ctx, "存档文件夹已清空")
	}

	err = os.RemoveAll(saveDataDir)
	if err != nil {
		runtime.LogErrorf(a.ctx, "清空数据文件夹失败: %v", err)
	} else {
		runtime.LogInfo(a.ctx, "数据文件夹已清空")
	}
}

func (a *App) ReadSavedDataFileName() ([]string, error) {
	// 获取当前运行的 exe 文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取程序路径失败: %w", err)
	}
	// 获取 exe 所在的目录
	exeDir := filepath.Dir(exePath)

	saveAllDir := filepath.Join(exeDir, "CSIFrame")
	entries, err := os.ReadDir(saveAllDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil // 或者返回 nil, nil，取决于前端期望收到空数组还是 null
		}
		return nil, fmt.Errorf("无法读取文件夹: %w", err)
	}
	var fileNames []string
	for _, entry := range entries {
		// 确保是文件而不是子文件夹，且后缀是 .json
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			fileNames = append(fileNames, entry.Name())
		}
	}
	return fileNames, nil
}

func (a *App) LoadFrameFile(filename string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	saveAllDir := filepath.Join(exeDir, "CSIFrame")
	finalPath := filepath.Join(saveAllDir, filename)
	data, err := os.ReadFile(finalPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	var frames []*CSIFrame
	err = json.Unmarshal(data, &frames)
	if err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}
	for _, frame := range frames {
		runtime.EventsEmit(a.ctx, "load-csi-data", frame)
		time.Sleep(2 * time.Millisecond) // 模拟数据流入的间隔
	}
	return nil
}

func (a *App) SaveDataSegment(startIdx int, endIdx int, srcFileName string, newfilename string) error {
	fmt.Printf("准备保存数据段: start=%d, end=%d, srcFile=%s, newFile=%s\n", startIdx, endIdx, srcFileName, newfilename)
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	saveAllDir := filepath.Join(exeDir, "CSIFrame")
	saveDataDir := filepath.Join(exeDir, "Data")
	srcPath := filepath.Join(saveAllDir, srcFileName)
	newAllPath := filepath.Join(saveAllDir, newfilename)
	newDataPath := filepath.Join(saveDataDir, newfilename)

	fileBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("读取源文件失败: %v", err)
	}
	var fullData []*CSIFrame
	if err := json.Unmarshal(fileBytes, &fullData); err != nil {
		return fmt.Errorf("解析JSON失败: %v", err)
	}

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(fullData) {
		endIdx = len(fullData)
	}
	if startIdx >= endIdx {
		return fmt.Errorf("非法的截取范围: start=%d, end=%d", startIdx, endIdx)
	}
	segmentData := fullData[startIdx:endIdx]
	justRaws := make([][]int16, len(segmentData))
	for i, frame := range segmentData {
		if frame != nil {
			justRaws[i] = frame.Raw
		}
	}

	newJsonAll, err := json.Marshal(segmentData)
	if err != nil {
		return fmt.Errorf("序列化新数据失败: %v", err)
	}
	err = os.WriteFile(newAllPath, newJsonAll, 0644)
	if err != nil {
		return fmt.Errorf("写入新文件失败: %v", err)
	}

	newJsonData, err := json.Marshal(justRaws)
	if err != nil {
		return fmt.Errorf("序列化新数据失败: %v", err)
	}
	err = os.WriteFile(newDataPath, newJsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入新文件失败: %v", err)
	}
	return nil
}
