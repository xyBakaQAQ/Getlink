package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Config struct {
	Debug           bool   `json:"debug"`
	AutoConnect     bool   `json:"auto_connect"` // New field for automatic connection
	AutoConnectIP   string `json:"auto_connect_ip"`
	AutoConnectPort string `json:"auto_connect_port"`
}

var config = Config{
	Debug:           false,
	AutoConnect:     false, // Default to false
	AutoConnectIP:   "127.0.0.1",
	AutoConnectPort: "16384",
}

func loadConfig() {
	if file, err := os.Open("config.json"); err == nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			fmt.Println("[ERROR] 读取 config.json 失败:", err)
		}
	}
}

func debugPrint(v ...interface{}) {
	if config.Debug {
		fmt.Println(v...)
	}
}

func waitForExit() {
	// Removed from config, now hardcoded
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func listDevices() ([]string, error) {
	output, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return nil, err
	}
	lines, devices := strings.Split(string(output), "\n"), []string{}
	for _, line := range lines[1:] {
		if strings.Contains(line, "device") && !strings.Contains(line, "unauthorized") {
			devices = append(devices, strings.Fields(line)[0])
		}
	}
	debugPrint("[DEBUG] 设备列表:", devices)
	return devices, nil
}

func connectToADB() error {
	output, err := exec.Command("adb", "connect", fmt.Sprintf("%s:%s", config.AutoConnectIP, config.AutoConnectPort)).Output()
	if err != nil || !strings.Contains(string(output), "connected to") {
		return fmt.Errorf("无法连接到设备")
	}
	debugPrint("[DEBUG] adb connect 输出:\n", string(output))
	return nil
}

func selectDevice(devices []string) (string, error) {
	if len(devices) == 1 {
		return devices[0], nil
	}
	fmt.Println("检测到多个设备，请输入编号选择设备:")
	for i, device := range devices {
		fmt.Printf("[%d] %s\n", i+1, device)
	}
	var choice int
	fmt.Print("输入设备编号 (1 到 ", len(devices), "): ")
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(devices) {
		return "", fmt.Errorf("无效的选择")
	}
	return devices[choice-1], nil
}

func extractURLs(deviceID string) {
	cmd := exec.Command("adb", "-s", deviceID, "shell", "logcat")
	stdout, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		fmt.Println("[ERROR] 启动 adb logcat 失败:", err)
		waitForExit()
		return
	}
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)

	// 定义正则表达式
	ysURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/hk4e/event/[^\s]+`)  // 原神
	starRailURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/hkrpg/[^\s]+`) // 崩铁
	zzzURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/nap/event/[^\s]+`)  // 绝区零

	for scanner.Scan() {
		line := scanner.Text()
		if url := ysURLRegex.FindString(line); url != "" {
			fmt.Println("[原神] 找到的URL:", url)
			waitForExit()
			return
		}
		if url := starRailURLRegex.FindString(line); url != "" {
			fmt.Println("[崩坏：星穹铁道] 找到的URL:", url)
			waitForExit()
			return
		}
		if url := zzzURLRegex.FindString(line); url != "" {
			fmt.Println("[绝区零] 找到的URL:", url)
			waitForExit()
			return
		}
	}
	fmt.Println("未找到符合条件的URL")
	waitForExit()
}

func main() {
	loadConfig()
	fmt.Println("正在检查 ADB 连接状态...")

	var devices []string
	var err error

	// Check if automatic connection is enabled
	if config.AutoConnect {
		fmt.Printf("自动连接到设备 %s:%s...\n", config.AutoConnectIP, config.AutoConnectPort)
		if err := connectToADB(); err != nil {
			fmt.Println("[ERROR] 无法连接到设备:", err)
			waitForExit()
			return
		}
		devices, err = listDevices()
	} else {
		devices, err = listDevices()
	}

	if err != nil || len(devices) == 0 {
		fmt.Println("未检测到设备，程序将退出。")
		waitForExit()
		return
	}

	if deviceID, err := selectDevice(devices); err == nil {
		fmt.Println("正在监听日志，请打开抽卡界面...")
		extractURLs(deviceID)
	} else {
		fmt.Println("[ERROR] 设备选择错误:", err)
		waitForExit()
	}
}
