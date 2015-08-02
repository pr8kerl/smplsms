package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"time"
)

// Binding from JSON
type SMS struct {
	Mobile  string `form:"mobile" json:"mobile" binding:"required"`
	Message string `form:"message" json:"message" binding:"required"`
}

var (
	modem       *GSMModem
	msgs        chan SMS
	bindaddress string
)

func init() {

	// setup config
	err := InitialiseConfig(cfgfile)
	if err != nil {
		fmt.Printf("error reading config: %s\r\n", err)
	}
	bindaddress = fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort)
	modem = NewModem(config.CommPort, config.Baud, "Modem")
	msgs = make(chan SMS, config.BufferSize)
	log.SetFlags(log.LstdFlags)

}

func main() {

	err := modem.Connect()
	if err != nil {
		log.Printf("ConnectModem: error connecting to %s, %s\r\n", modem.DeviceId, err)
		log.Printf("commport: %s\r\n", config.CommPort)
		log.Printf("baud: %d\r\n", config.Baud)
		os.Exit(1)
	}
	err = modem.InitialiseModem()
	if err != nil {
		log.Printf("InitModem: error initialising %s, %s\r\n", modem.DeviceId, err)
	}

	// Creates a router without any middleware by default
	//gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middlewares
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(SetJellyBeans())

	api := r.Group("/api")
	{
		api.GET("/sms", index)
		api.POST("/sms", sendSMS)
	}

	go worker()
	// Listen and server on 0.0.0.0:8951
	r.Run(bindaddress)

}

func index(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": 200, "message": "hello"})
}

func SetJellyBeans() gin.HandlerFunc {
	// Do some initialization logic here
	// Foo()
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Powered-By", "Black Jelly Beans")
		c.Next()
	}
}

func sendSMS(c *gin.Context) {
	var json SMS
	if c.BindJSON(&json) == nil {
		if json.Mobile == "" || json.Message == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"status": 500, "message": "invalid message format"})
		} else {
			msgs <- json
			c.JSON(http.StatusOK, gin.H{"status": 200, "message": "message received"})
		}
	}
}

func worker() {

	for {
		m := <-msgs
		log.Printf("msg received: %s\r\n", m)
		time.Sleep(time.Second)

		err := modem.SendSMS(m.Mobile, m.Message)
		if err != nil {
			log.Printf("msg error: %s\r\n", err)
			log.Printf("msg failure for msg: %s\r\n", m)
		} else {
			log.Printf("msg success: %s\r\n", m)
		}

	}

}
