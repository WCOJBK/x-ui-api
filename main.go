package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"x-ui/config"
	"x-ui/database"
	"x-ui/logger"
	"x-ui/sub"
	"x-ui/web"
	"x-ui/web/global"
	"x-ui/web/service"

	"github.com/op/go-logging"
)

// runWebServer starts the web panel and subscription server
func runWebServer() {
	log.Printf("Starting %v %v", config.GetName(), config.GetVersion())

	// Initialize logger based on config
	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Notice:
		logger.InitLogger(logging.NOTICE)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatalf("Unknown log level: %v", config.GetLogLevel())
	}

	// Initialize database
	if err := database.InitDB(config.GetDBPath()); err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	// Start web server
	server := web.NewServer()
	global.SetWebServer(server)
	if err := server.Start(); err != nil {
		log.Fatalf("Error starting web server: %v", err)
		return
	}

	// Start subscription server
	subServer := sub.NewServer()
	global.SetSubServer(subServer)
	if err := subServer.Start(); err != nil {
		log.Fatalf("Error starting sub server: %v", err)
		return
	}

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			logger.Info("Received SIGHUP signal. Restarting servers...")
			
			if err := server.Stop(); err != nil {
				logger.Debug("Error stopping web server:", err)
			}
			if err := subServer.Stop(); err != nil {
				logger.Debug("Error stopping sub server:", err)
			}

			server = web.NewServer()
			global.SetWebServer(server)
			if err := server.Start(); err != nil {
				log.Fatalf("Error restarting web server: %v", err)
				return
			}
			log.Println("Web server restarted successfully.")

			subServer = sub.NewServer()
			global.SetSubServer(subServer)
			if err := subServer.Start(); err != nil {
				log.Fatalf("Error restarting sub server: %v", err)
				return
			}
			log.Println("Sub server restarted successfully.")

		default:
			server.Stop()
			subServer.Stop()
			log.Println("Shutting down servers.")
			return
		}
	}
}

// resetSetting resets all panel settings to default values
func resetSetting() {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("Failed to initialize database:", err)
		return
	}

	settingService := service.SettingService{}
	if err := settingService.ResetSettings(); err != nil {
		fmt.Println("Failed to reset settings:", err)
	} else {
		fmt.Println("Settings successfully reset.")
	}
}

// showSetting displays current panel settings if show is true
func showSetting(show bool) {
	if !show {
		return
	}

	settingService := service.SettingService{}
	userService := service.UserService{}

	// Get port
	port, err := settingService.GetPort()
	if err != nil {
		fmt.Println("Get current port failed:", err)
	}

	// Get web base path
	webBasePath, err := settingService.GetBasePath()
	if err != nil {
		fmt.Println("Get webBasePath failed:", err)
	}

	// Get SSL certificate settings
	certFile, err := settingService.GetCertFile()
	if err != nil {
		fmt.Println("Get cert file failed:", err)
	}
	keyFile, err := settingService.GetKeyFile()
	if err != nil {
		fmt.Println("Get key file failed:", err)
	}

	// Get user credentials
	userModel, err := userService.GetFirstUser()
	if err != nil {
		fmt.Println("Get current user info failed:", err)
	}

	// Display settings
	fmt.Println("Current panel settings:")
	if certFile == "" || keyFile == "" {
		fmt.Println("Warning: Panel is not secure with SSL")
	} else {
		fmt.Println("Panel is secure with SSL")
	}
	
	if userModel.Username == "" || userModel.Password == "" {
		fmt.Println("Warning: Current username or password is empty")
	} else {
		fmt.Println("Username:", userModel.Username)
		fmt.Println("Password:", userModel.Password)
	}
	
	fmt.Println("Port:", port)
	fmt.Println("Web Base Path:", webBasePath)
}

// updateTgbotEnableSts updates Telegram bot enabled status
func updateTgbotEnableSts(status bool) {
	settingService := service.SettingService{}
	
	currentStatus, err := settingService.GetTgbotEnabled()
	if err != nil {
		fmt.Println("Error getting current Telegram bot status:", err)
		return
	}

	logger.Infof("Current Telegram bot status: %v, updating to: %v", currentStatus, status)
	
	if currentStatus != status {
		if err := settingService.SetTgbotEnabled(status); err != nil {
			fmt.Println("Error updating Telegram bot status:", err)
		} else {
			logger.Infof("Successfully updated Telegram bot status to: %v", status)
		}
	}
}

// updateTgbotSetting updates Telegram bot settings
func updateTgbotSetting(tgBotToken, tgBotChatid, tgBotRuntime string) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	settingService := service.SettingService{}

	if tgBotToken != "" {
		if err := settingService.SetTgBotToken(tgBotToken); err != nil {
			fmt.Printf("Error setting Telegram bot token: %v\n", err)
			return
		}
		logger.Info("Successfully updated Telegram bot token")
	}

	if tgBotRuntime != "" {
		if err := settingService.SetTgbotRuntime(tgBotRuntime); err != nil {
			fmt.Printf("Error setting Telegram bot runtime: %v\n", err)
			return
		}
		logger.Infof("Successfully updated Telegram bot runtime to: %s", tgBotRuntime)
	}

	if tgBotChatid != "" {
		if err := settingService.SetTgBotChatId(tgBotChatid); err != nil {
			fmt.Printf("Error setting Telegram bot chat ID: %v\n", err)
			return
		}
		logger.Info("Successfully updated Telegram bot chat ID")
	}
}

// updateSetting updates panel settings
func updateSetting(port int, username, password, webBasePath, listenIP string) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("Database initialization failed:", err)
		return
	}

	settingService := service.SettingService{}
	userService := service.UserService{}

	if port > 0 {
		if err := settingService.SetPort(port); err != nil {
			fmt.Println("Failed to set port:", err)
		} else {
			fmt.Printf("Port set successfully: %v\n", port)
		}
	}

	if username != "" || password != "" {
		if err := userService.UpdateFirstUser(username, password); err != nil {
			fmt.Println("Failed to update username and password:", err)
		} else {
			fmt.Println("Username and password updated successfully")
		}
	}

	if webBasePath != "" {
		if err := settingService.SetBasePath(webBasePath); err != nil {
			fmt.Println("Failed to set base URI path:", err)
		} else {
			fmt.Println("Base URI path set successfully")
		}
	}

	if listenIP != "" {
		if err := settingService.SetListen(listenIP); err != nil {
			fmt.Println("Failed to set listen IP:", err)
		} else {
			fmt.Printf("Listen IP %v set successfully\n", listenIP)
		}
	}
}

// updateCert updates SSL certificate settings
func updateCert(publicKey, privateKey string) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("Database initialization failed:", err)
		return
	}

	if (privateKey != "" && publicKey != "") || (privateKey == "" && publicKey == "") {
		settingService := service.SettingService{}
		
		if err := settingService.SetCertFile(publicKey); err != nil {
			fmt.Println("Failed to set certificate public key:", err)
		} else {
			fmt.Println("Certificate public key set successfully")
		}

		if err := settingService.SetKeyFile(privateKey); err != nil {
			fmt.Println("Failed to set certificate private key:", err)
		} else {
			fmt.Println("Certificate private key set successfully")
		}
	} else {
		fmt.Println("Error: Both public and private keys must be provided")
	}
}

// GetCertificate displays current SSL certificate settings
func GetCertificate(getCert bool) {
	if !getCert {
		return
	}

	settingService := service.SettingService{}
	
	certFile, err := settingService.GetCertFile()
	if err != nil {
		fmt.Println("Failed to get certificate file:", err)
	}
	
	keyFile, err := settingService.GetKeyFile()
	if err != nil {
		fmt.Println("Failed to get key file:", err)
	}

	fmt.Println("Certificate:", certFile)
	fmt.Println("Key:", keyFile)
}

// GetListenIP displays current listen IP setting
func GetListenIP(getListen bool) {
	if !getListen {
		return
	}

	settingService := service.SettingService{}
	listenIP, err := settingService.GetListen()
	if err != nil {
		fmt.Println("Failed to get listen IP:", err)
		return
	}

	fmt.Println("Listen IP:", listenIP)
}

// migrateDb performs database migration
func migrateDb() {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		log.Fatal("Database initialization failed:", err)
	}

	fmt.Println("Starting database migration...")
	inboundService := service.InboundService{}
	inboundService.MigrateDB()
	fmt.Println("Migration completed successfully!")
}

// removeSecret removes user secret and updates settings
func removeSecret() {
	userService := service.UserService{}
	settingService := service.SettingService{}

	secretExists, err := userService.CheckSecretExistence()
	if err != nil {
		fmt.Println("Error checking secret existence:", err)
		return
	}

	if !secretExists {
		fmt.Println("No secret exists to remove")
		return
	}

	if err := userService.RemoveUserSecret(); err != nil {
		fmt.Println("Error removing secret:", err)
		return
	}

	if err := settingService.SetSecretStatus(false); err != nil {
		fmt.Println("Error updating secret status:", err)
		return
	}

	fmt.Println("Secret removed successfully")
}

func main() {
	if len(os.Args) < 2 {
		runWebServer()
		return
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Show version")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)

	// Setting command flags
	var (
		port          int
		username      string
		password      string
		webBasePath   string
		listenIP      string
		getListen     bool
		webCertFile   string
		webKeyFile    string
		tgbottoken    string
		tgbotchatid   string
		enabletgbot   bool
		tgbotRuntime  string
		reset         bool
		show          bool
		getCert       bool
		removeSecret  bool
	)

	settingCmd.BoolVar(&reset, "reset", false, "Reset all settings")
	settingCmd.BoolVar(&show, "show", false, "Display current settings")
	settingCmd.BoolVar(&removeSecret, "remove_secret", false, "Remove secret key")
	settingCmd.IntVar(&port, "port", 0, "Set panel port number")
	settingCmd.StringVar(&username, "username", "", "Set login username")
	settingCmd.StringVar(&password, "password", "", "Set login password")
	settingCmd.StringVar(&webBasePath, "webBasePath", "", "Set base path for Panel")
	settingCmd.StringVar(&listenIP, "listenIP", "", "Set panel listen IP")
	settingCmd.BoolVar(&getListen, "getListen", false, "Display current panel listen IP")
	settingCmd.BoolVar(&getCert, "getCert", false, "Display current certificate settings")
	settingCmd.StringVar(&webCertFile, "webCert", "", "Set path to public key file for panel")
	settingCmd.StringVar(&webKeyFile, "webCertKey", "", "Set path to private key file for panel")
	settingCmd.StringVar(&tgbottoken, "tgbottoken", "", "Set token for Telegram bot")
	settingCmd.StringVar(&tgbotRuntime, "tgbotRuntime", "", "Set cron time for Telegram bot notifications")
	settingCmd.StringVar(&tgbotchatid, "tgbotchatid", "", "Set chat ID for Telegram bot notifications")
	settingCmd.BoolVar(&enabletgbot, "enabletgbot", false, "Enable notifications via Telegram bot")

	// Custom usage information
	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    run        Run web panel")
		fmt.Println("    migrate    Migrate from other/old x-ui")
		fmt.Println("    setting    Configure settings")
	}

	flag.Parse()

	if showVersion {
		fmt.Println(config.GetVersion())
		return
	}

	switch os.Args[1] {
	case "run":
		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing run command:", err)
			return
		}
		runWebServer()

	case "migrate":
		migrateDb()

	case "setting":
		if err := settingCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing setting command:", err)
			return
		}

		if reset {
			resetSetting()
		} else {
			updateSetting(port, username, password, webBasePath, listenIP)
		}

		if show {
			showSetting(show)
		}
		if getListen {
			GetListenIP(getListen)
		}
		if getCert {
			GetCertificate(getCert)
		}
		if tgbottoken != "" || tgbotchatid != "" || tgbotRuntime != "" {
			updateTgbotSetting(tgbottoken, tgbotchatid, tgbotRuntime)
		}
		if removeSecret {
			removeSecret()
		}
		if enabletgbot {
			updateTgbotEnableSts(enabletgbot)
		}

	case "cert":
		if err := settingCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing cert command:", err)
			return
		}

		if reset {
			updateCert("", "")
		} else {
			updateCert(webCertFile, webKeyFile)
		}

	default:
		fmt.Println("Invalid command")
		fmt.Println()
		runCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}
}
