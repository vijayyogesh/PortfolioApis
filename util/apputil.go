package util

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
	"github.com/vijayyogesh/PortfolioApis/constants"
)

var Logger *log.Logger

type AppUtil struct {
	Db        *sql.DB
	AppLogger *log.Logger
	Config    *Config
}

type Config struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBPort     int    `mapstructure:"DB_PORT"`
	APPPort    int    `mapstructure:"APP_PORT"`
}

/* Initialize/Create AppLevel/Global objects
Terminates if any one of it fails */
func NewAppUtil() *AppUtil {
	InitializeLog()
	config := LoadConfig(constants.AppEnvPath)
	db := SetupDB(config)

	Logger.Println("Completed NewAppUtil")
	Logger.Println(" --- INITIALIZATION SUCCESSFULL --- ")

	return &AppUtil{
		db,
		Logger,
		config,
	}
}

/* Log error and EXIT COMPLETELY. Do not use unless program needs to be terminated. */
func handleCriticalErr(err error) {
	if err != nil {
		Logger.Fatal(err)
	}
}

/* Initialize App Level Logger */
func InitializeLog() {
	/* If the file doesn't exist, create it or append to the file */
	file, err := os.OpenFile(constants.AppLoggerFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	handleCriticalErr(err)

	Logger = log.New(file, constants.AppLoggerPrefix, log.Ldate|log.Ltime|log.Lshortfile)
	Logger.SetOutput(file)
	Logger.Println("Completed InitializeLog File with FileName - " + constants.AppLoggerFile + " ,Prefix - " + constants.AppLoggerPrefix)
}

/* Import config from env file */
func LoadConfig(path string) (config *Config) {
	Logger.Println("Starting LoadConfig")

	viper.AddConfigPath(path)
	viper.SetConfigName(constants.AppEnvName)
	viper.SetConfigType(constants.AppEnvType)

	viper.AutomaticEnv()

	errReadConfig := viper.ReadInConfig()
	handleCriticalErr(errReadConfig)

	errUnmarshal := viper.Unmarshal(&config)
	handleCriticalErr(errUnmarshal)

	Logger.Printf("ENV FILE VALUES - host=%s port=%d user=%s dbname=%s dbdriver=%s", config.DBHost, config.DBPort, config.DBUser, config.DBName, config.DBDriver)
	Logger.Println("Completed LoadConfig")
	return config
}

/* Setup DB Connection */
func SetupDB(config *Config) (db *sql.DB) {
	Logger.Println("Starting SetupDB")
	dbinfo := fmt.Sprintf(constants.AppDBFmtString, config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := sql.Open(config.DBDriver, dbinfo)
	handleCriticalErr(err)

	db.SetMaxOpenConns(constants.AppDBMaxConn)

	Logger.Println("Completed SetupDB")
	return db
}
