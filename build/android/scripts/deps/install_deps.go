// Package main 检查 Android 开发依赖。CI 非空或 TASK_FORCE_YES=true 时，创建 AVD 的提示自动接受。
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// main 检查 Go、ANDROID_HOME、adb、模拟器、NDK、Java，并在没有 AVD 时提议用已安装的系统镜像创建。有缺失项时以状态 1 退出。
func main() {
	fmt.Println("Checking Android development dependencies...")
	fmt.Println()

	errors := []string{}

	// 检查 Go。
	if !checkCommand("go", "version") {
		errors = append(errors, "Go is not installed. Install from https://go.dev/dl/")
	} else {
		fmt.Println("✓ Go is installed")
	}

	// 检查 ANDROID_HOME，空则再看 ANDROID_SDK_ROOT。
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		androidHome = os.Getenv("ANDROID_SDK_ROOT")
	}
	if androidHome == "" {
		// 再试常见的默认 SDK 路径。
		home, _ := os.UserHomeDir()
		possiblePaths := []string{
			filepath.Join(home, "Android", "Sdk"),
			filepath.Join(home, "Library", "Android", "sdk"),
			"/usr/local/share/android-sdk",
		}
		for _, p := range possiblePaths {
			if _, err := os.Stat(p); err == nil {
				androidHome = p
				break
			}
		}
	}

	if androidHome == "" {
		errors = append(errors, "ANDROID_HOME not set. Install Android Studio and set ANDROID_HOME environment variable")
	} else {
		fmt.Printf("✓ ANDROID_HOME: %s\n", androidHome)
	}

	// 检查 adb。
	if !checkCommand("adb", "version") {
		if androidHome != "" {
			platformTools := filepath.Join(androidHome, "platform-tools")
			errors = append(errors, fmt.Sprintf("adb not found. Add %s to PATH", platformTools))
		} else {
			errors = append(errors, "adb not found. Install Android SDK Platform-Tools")
		}
	} else {
		fmt.Println("✓ adb is installed")
	}

	// 检查 emulator。
	if !checkCommand("emulator", "-list-avds") {
		if androidHome != "" {
			emulatorPath := filepath.Join(androidHome, "emulator")
			errors = append(errors, fmt.Sprintf("emulator not found. Add %s to PATH", emulatorPath))
		} else {
			errors = append(errors, "emulator not found. Install Android Emulator via SDK Manager")
		}
	} else {
		fmt.Println("✓ Android Emulator is installed")
	}

	// 检查 NDK。ANDROID_NDK_HOME 为空时，用 SDK 下 ndk 目录里的第一个子目录。
	ndkHome := os.Getenv("ANDROID_NDK_HOME")
	if ndkHome == "" && androidHome != "" {
		// 在默认的 ndk 目录里找第一份。
		ndkDir := filepath.Join(androidHome, "ndk")
		if entries, err := os.ReadDir(ndkDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					ndkHome = filepath.Join(ndkDir, entry.Name())
					break
				}
			}
		}
	}

	if ndkHome == "" {
		errors = append(errors, "Android NDK not found. Install NDK via Android Studio > SDK Manager > SDK Tools > NDK (Side by side)")
	} else {
		fmt.Printf("✓ Android NDK: %s\n", ndkHome)
	}

	// 检查 Java。
	if !checkCommand("java", "-version") {
		errors = append(errors, "Java not found. Install JDK 11+ (OpenJDK recommended)")
	} else {
		fmt.Println("✓ Java is installed")
	}

	// 能列出 AVD 且输出非空就计数。列表为空时才提议创建，并且只用已经安装的系统镜像。
	if checkCommand("emulator", "-list-avds") {
		cmd := exec.Command("emulator", "-list-avds")
		output, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(output))) > 0 {
			avds := strings.Split(strings.TrimSpace(string(output)), "\n")
			fmt.Printf("✓ Found %d Android Virtual Device(s)\n", len(avds))
		} else {
			// 没有 AVD 时提议创建。只使用已经装好的系统镜像，这里不运行 sdkmanager。
			offerCreateAVD(androidHome)
		}
	}

	fmt.Println()

	if len(errors) > 0 {
		fmt.Println("❌ Missing dependencies:")
		for _, err := range errors {
			fmt.Printf("   - %s\n", err)
		}
		fmt.Println()
		fmt.Println("Setup instructions:")
		fmt.Println("1. Install Android Studio: https://developer.android.com/studio")
		fmt.Println("2. Open SDK Manager and install:")
		fmt.Println("   - Android SDK Platform (API 35)")
		fmt.Println("   - Android SDK Build-Tools")
		fmt.Println("   - Android SDK Platform-Tools")
		fmt.Println("   - Android Emulator")
		fmt.Println("   - NDK (Side by side)")
		fmt.Println("3. Set environment variables:")
		if runtime.GOOS == "darwin" {
			fmt.Println("   export ANDROID_HOME=$HOME/Library/Android/sdk")
		} else {
			fmt.Println("   export ANDROID_HOME=$HOME/Android/Sdk")
		}
		fmt.Println("   export PATH=$PATH:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator")
		fmt.Println("4. Create an AVD via Android Studio > Tools > Device Manager")
		os.Exit(1)
	}

	fmt.Println("✓ All Android development dependencies are installed!")
}

// checkCommand 运行 name 和 args。标准输出和标准错误都丢掉，退出码为 0 才返回 true。
func checkCommand(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// offerCreateAVD 在没有 AVD 时提议创建名为 wails 的设备。只从已经安装、且 ABI 与本机匹配的系统镜像里选 API 数字最大的一份。这里不运行 sdkmanager。
// androidHome 非空时用来 Glob system-images 下匹配 ABI 的镜像，并传给 findAVDManager。选中的镜像路径也用它去掉 system-images 前缀来拼包名。
func offerCreateAVD(androidHome string) {
	abi := "x86_64"
	if runtime.GOARCH == "arm64" {
		abi = "arm64-v8a"
	}

	// 在已安装、ABI 匹配的系统镜像里取 API 数字最大的一份。按数字比，避免 android-9 排在 android-35 前面。
	var img string
	if androidHome != "" {
		matches, _ := filepath.Glob(filepath.Join(androidHome, "system-images", "android-*", "*", abi))
		bestAPI := -1
		for _, m := range matches {
			apiDir := filepath.Base(filepath.Dir(filepath.Dir(m)))
			api, err := strconv.Atoi(strings.TrimPrefix(apiDir, "android-"))
			if err != nil {
				continue // 预览或扩展镜像，例如 android-35-ext14，Atoi 会失败。
			}
			if api > bestAPI {
				bestAPI = api
				img = m
			}
		}
	}

	avdmanager := findAVDManager(androidHome)

	if img == "" || avdmanager == "" {
		fmt.Println("⚠ No Android Virtual Devices found.")
		fmt.Println("   Install a system image and create an AVD, e.g.:")
		fmt.Printf("     sdkmanager 'system-images;android-35;google_apis;%s'\n", abi)
		fmt.Printf("     avdmanager create avd --name wails --package 'system-images;android-35;google_apis;%s' --device pixel_7\n", abi)
		return
	}

	// 从已安装镜像目录推出包名。例如 <sdk>/system-images/android-35/google_apis/arm64-v8a 变成 system-images;android-35;google_apis;arm64-v8a。
	rel := strings.TrimPrefix(img, filepath.Join(androidHome, "system-images")+string(os.PathSeparator))
	pkg := "system-images;" + strings.ReplaceAll(rel, string(os.PathSeparator), ";")

	fmt.Println("⚠ No Android Virtual Devices found.")
	fmt.Printf("   Would you like to create a 'wails' AVD from %s?\n", pkg)
	if !promptUser("Create AVD?") {
		fmt.Println("   Skipping AVD creation.")
		fmt.Printf("   Create manually: avdmanager create avd --name wails --package '%s' --device pixel_7\n", pkg)
		return
	}

	cmd := exec.Command(avdmanager, "create", "avd", "--name", "wails", "--package", pkg, "--device", "pixel_7", "--force")
	cmd.Stdin = strings.NewReader("no\n") // 拒绝自定义硬件配置的提示。
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("   Failed to create AVD: %v\n", err)
	} else {
		fmt.Println("   ✅ 'wails' AVD created")
	}
}

// findAVDManager 先在 PATH 里找 avdmanager。没有则在 SDK 的 cmdline-tools/*/bin 里找。目录名是 latest 的优先。否则按版本数字取最大的。都没有就返回空字符串。
// androidHome 非空且 PATH 里没有 avdmanager 时，用来 Glob cmdline-tools/*/bin/avdmanager。为空时在 PATH 也没有的情况下返回空字符串。
func findAVDManager(androidHome string) string {
	if p, err := exec.LookPath("avdmanager"); err == nil {
		return p
	}
	if androidHome != "" {
		matches, _ := filepath.Glob(filepath.Join(androidHome, "cmdline-tools", "*", "bin", "avdmanager"))
		// 目录名 latest 优先。其余按数字比版本，避免 9.0 排在 11.0 前面。解析失败的版本当成 0。
		best := ""
		bestVersion := -1.0
		for _, m := range matches {
			version := filepath.Base(filepath.Dir(filepath.Dir(m)))
			if version == "latest" {
				return m
			}
			v, err := strconv.ParseFloat(version, 64)
			if err != nil {
				v = 0
			}
			if v > bestVersion {
				bestVersion = v
				best = m
			}
		}
		return best
	}
	return ""
}

// promptUser 在 CI 非空或 TASK_FORCE_YES 等于 true 时返回 true。否则读一行，小写后是 y 或 yes 才返回 true。读失败返回 false。
// question 写进提示，后面接 [y/N]:。
func promptUser(question string) bool {
	if os.Getenv("CI") != "" || os.Getenv("TASK_FORCE_YES") == "true" {
		fmt.Printf("%s [y/N]: y (auto-accepted)\n", question)
		return true
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", question)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
