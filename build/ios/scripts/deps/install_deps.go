// Package main 检查 iOS 开发依赖，用 Go 而不是 shell，以便换 shell 也能跑。
// 直接 go run 是交互模式。TASK_FORCE_YES=true 或 CI 非空时自动接受提示。
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Dependency 是一项要检查的 iOS 开发依赖。
type Dependency struct {
	// Name 是显示名。
	Name string
	// CheckFunc 返回是否通过，以及补充说明。
	CheckFunc func() (bool, string)
	// Required 为真且检查失败时，最后以状态 1 退出。
	Required bool
	// InstallCmd 非空时可以提示用户执行。
	InstallCmd []string
	// InstallMsg 是失败时给用户的安装说明。
	InstallMsg string
	// SuccessMsg 是通过时打印的句子。
	SuccessMsg string
	// FailureMsg 是失败时打印的句子。
	FailureMsg string
}

// main 依次检查 Xcode、开发者目录、iOS SDK、模拟器运行时，再看有没有 iPhone 模拟器。有必需项失败时以状态 1 退出。
func main() {
	fmt.Println("Checking iOS development dependencies...")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	hasErrors := false
	dependencies := []Dependency{
		{
			Name: "Xcode",
			CheckFunc: func() (bool, string) {
				// 看 xcodebuild 是否存在。
				if !checkCommand([]string{"xcodebuild", "-version"}) {
					return false, ""
				}
				// 取版本信息。
				out, err := exec.Command("xcodebuild", "-version").Output()
				if err != nil {
					return false, ""
				}
				lines := strings.Split(string(out), "\n")
				if len(lines) > 0 {
					return true, strings.TrimSpace(lines[0])
				}
				return true, ""
			},
			Required:   true,
			InstallMsg: "Please install Xcode from the Mac App Store:\n   https://apps.apple.com/app/xcode/id497799835\n   Xcode is REQUIRED for iOS development (includes iOS SDKs, simulators, and frameworks)",
			SuccessMsg: "✅ Xcode found",
			FailureMsg: "❌ Xcode not found (REQUIRED)",
		},
		{
			Name: "Xcode Developer Path",
			CheckFunc: func() (bool, string) {
				// 看 xcode-select 是否指向有效的 Xcode 路径。
				out, err := exec.Command("xcode-select", "-p").Output()
				if err != nil {
					return false, "xcode-select not configured"
				}
				path := strings.TrimSpace(string(out))

				// 看路径是否存在，并且落在 Xcode.app 里。
				if _, err := os.Stat(path); err != nil {
					return false, "Invalid Xcode path"
				}

				// 确认指向的是 Xcode.app，而不是只有命令行工具。
				if !strings.Contains(path, "Xcode.app") {
					return false, fmt.Sprintf("Points to %s (should be Xcode.app)", path)
				}

				return true, path
			},
			Required:   true,
			InstallCmd: []string{"sudo", "xcode-select", "-s", "/Applications/Xcode.app/Contents/Developer"},
			InstallMsg: "Xcode developer path needs to be configured",
			SuccessMsg: "✅ Xcode developer path configured",
			FailureMsg: "❌ Xcode developer path not configured correctly",
		},
		{
			Name: "iOS SDK",
			CheckFunc: func() (bool, string) {
				// 取 iOS 模拟器 SDK 路径。
				cmd := exec.Command("xcrun", "--sdk", "iphonesimulator", "--show-sdk-path")
				output, err := cmd.Output()
				if err != nil {
					return false, "Cannot find iOS SDK"
				}
				sdkPath := strings.TrimSpace(string(output))

				// 看 SDK 路径是否存在。
				if _, err := os.Stat(sdkPath); err != nil {
					return false, "iOS SDK path not found"
				}

				// 看 UIKit.framework 是否在。iOS 开发需要它。
				uikitPath := fmt.Sprintf("%s/System/Library/Frameworks/UIKit.framework", sdkPath)
				if _, err := os.Stat(uikitPath); err != nil {
					return false, "UIKit.framework not found"
				}

				// 取 SDK 版本。
				versionCmd := exec.Command("xcrun", "--sdk", "iphonesimulator", "--show-sdk-version")
				versionOut, _ := versionCmd.Output()
				version := strings.TrimSpace(string(versionOut))

				return true, fmt.Sprintf("iOS %s SDK", version)
			},
			Required:   true,
			InstallMsg: "iOS SDK comes with Xcode. Please ensure Xcode is properly installed.",
			SuccessMsg: "✅ iOS SDK found with UIKit framework",
			FailureMsg: "❌ iOS SDK not found or incomplete",
		},
		{
			Name: "iOS Simulator Runtime",
			CheckFunc: func() (bool, string) {
				if !checkCommand([]string{"xcrun", "simctl", "help"}) {
					return false, ""
				}
				// 看能不能列出运行时。
				out, err := exec.Command("xcrun", "simctl", "list", "runtimes").Output()
				if err != nil {
					return false, "Cannot access simulator"
				}
				// 数 iOS 运行时。
				lines := strings.Split(string(out), "\n")
				count := 0
				var versions []string
				for _, line := range lines {
					if strings.Contains(line, "iOS") && !strings.Contains(line, "unavailable") {
						count++
						// 抽出版本号。
						if parts := strings.Fields(line); len(parts) > 2 {
							for _, part := range parts {
								if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
									versions = append(versions, strings.Trim(part, "()"))
									break
								}
							}
						}
					}
				}
				if count > 0 {
					return true, fmt.Sprintf("%d runtime(s): %s", count, strings.Join(versions, ", "))
				}
				return false, "No iOS runtimes installed"
			},
			Required:   true,
			InstallMsg: "iOS Simulator runtimes come with Xcode. You may need to download them:\n   Xcode → Settings → Platforms → iOS",
			SuccessMsg: "✅ iOS Simulator runtime available",
			FailureMsg: "❌ iOS Simulator runtime not available",
		},
	}

	// 逐项检查。
	for _, dep := range dependencies {
		success, details := dep.CheckFunc()
		if success {
			msg := dep.SuccessMsg
			if details != "" {
				msg = fmt.Sprintf("%s (%s)", dep.SuccessMsg, details)
			}
			fmt.Println(msg)
		} else {
			fmt.Println(dep.FailureMsg)
			if details != "" {
				fmt.Printf("   Details: %s\n", details)
			}
			if dep.Required {
				hasErrors = true
				if len(dep.InstallCmd) > 0 {
					fmt.Println()
					fmt.Println("   " + dep.InstallMsg)
					fmt.Printf("   Fix command: %s\n", strings.Join(dep.InstallCmd, " "))
					if promptUser("Do you want to run this command?") {
						fmt.Println("Running command...")
						cmd := exec.Command(dep.InstallCmd[0], dep.InstallCmd[1:]...)
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Stdin = os.Stdin
						if err := cmd.Run(); err != nil {
							fmt.Printf("Command failed: %v\n", err)
							os.Exit(1)
						}
						fmt.Println("✅ Command completed. Please run this check again.")
					} else {
						fmt.Printf("   Please run manually: %s\n", strings.Join(dep.InstallCmd, " "))
					}
				} else {
					fmt.Println("   " + dep.InstallMsg)
				}
			}
		}
	}

	// 再看有没有 iPhone 模拟器设备。
	fmt.Println()
	fmt.Println("Checking for iPhone simulator devices...")
	if !checkCommand([]string{"xcrun", "simctl", "list", "devices"}) {
		fmt.Println("❌ Cannot check for iPhone simulators")
		hasErrors = true
	} else {
		out, err := exec.Command("xcrun", "simctl", "list", "devices").Output()
		if err != nil {
			fmt.Println("❌ Failed to list simulator devices")
			hasErrors = true
		} else if !strings.Contains(string(out), "iPhone") {
			fmt.Println("⚠️  No iPhone simulator devices found")
			fmt.Println()

			// 取列表里最后一条可用的 iOS 运行时。
			runtimeOut, err := exec.Command("xcrun", "simctl", "list", "runtimes").Output()
			if err != nil {
				fmt.Println("   Failed to get iOS runtimes:", err)
			} else {
				lines := strings.Split(string(runtimeOut), "\n")
				var latestRuntime string
				for _, line := range lines {
					if strings.Contains(line, "iOS") && !strings.Contains(line, "unavailable") {
						// 抽出运行时标识，用该行最后一个字段。
						parts := strings.Fields(line)
						if len(parts) > 0 {
							latestRuntime = parts[len(parts)-1]
						}
					}
				}

				if latestRuntime == "" {
					fmt.Println("   No iOS runtime found. Please install iOS simulators in Xcode:")
					fmt.Println("   Xcode → Settings → Platforms → iOS")
				} else {
					fmt.Println("   Would you like to create an iPhone 15 Pro simulator?")
					createCmd := []string{"xcrun", "simctl", "create", "iPhone 15 Pro", "iPhone 15 Pro", latestRuntime}
					fmt.Printf("   Command: %s\n", strings.Join(createCmd, " "))
					if promptUser("Create simulator?") {
						cmd := exec.Command(createCmd[0], createCmd[1:]...)
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						if err := cmd.Run(); err != nil {
							fmt.Printf("   Failed to create simulator: %v\n", err)
						} else {
							fmt.Println("   ✅ iPhone 15 Pro simulator created")
						}
					} else {
						fmt.Println("   Skipping simulator creation")
						fmt.Printf("   Create manually: %s\n", strings.Join(createCmd, " "))
					}
				}
			}
		} else {
			// 数名称里含 iPhone 且不是 unavailable 的设备。
			count := 0
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "iPhone") && !strings.Contains(line, "unavailable") {
					count++
				}
			}
			fmt.Printf("✅ %d iPhone simulator device(s) available\n", count)
		}
	}

	// 打印总结。有必需项失败就以状态 1 退出。
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 50))
	if hasErrors {
		fmt.Println("❌ Some required dependencies are missing or misconfigured.")
		fmt.Println()
		fmt.Println("Quick setup guide:")
		fmt.Println("1. Install Xcode from Mac App Store (if not installed)")
		fmt.Println("2. Open Xcode once and agree to the license")
		fmt.Println("3. Install additional components when prompted")
		fmt.Println("4. Run: sudo xcode-select -s /Applications/Xcode.app/Contents/Developer")
		fmt.Println("5. Download iOS simulators: Xcode → Settings → Platforms → iOS")
		fmt.Println("6. Run this check again")
		os.Exit(1)
	} else {
		fmt.Println("✅ All required dependencies are installed!")
		fmt.Println("   You're ready for iOS development with Wails!")
	}
}

// checkCommand 运行 args。参数为空返回 false。标准输出和标准错误都丢掉，退出码为 0 才返回 true。
func checkCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}

// promptUser 在 CI 非空或 TASK_FORCE_YES 等于 true 时直接返回 true。否则读一行，小写后是 y 或 yes 才返回 true。读失败返回 false。
func promptUser(question string) bool {
	// CI 非空或 TASK_FORCE_YES 为 true 时不再读标准输入。
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
