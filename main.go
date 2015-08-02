package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
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
	InitialiseConfig(cfgfile)
	bindaddress = fmt.Sprintf("%s:%d", config.BindAddress, config.BindPort)
	modem = NewModem(config.CommPort, config.Baud, "Modem")
	msgs = make(chan SMS, config.BufferSize)

}

func main() {

	err := modem.Connect()
	if err != nil {
		fmt.Printf("InitModem: error connecting to %s, %s\r\n", modem.DeviceId, err)
		os.Exit(1)
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
		fmt.Println("msg received: ", m)
		time.Sleep(time.Second)

		err := modem.SendSMS(m.Mobile, m.Message)
		if err != nil {
			fmt.Printf("msg error: %s\r\n", err)
			fmt.Printf("msg failure for msg: %s\r\n", m)
		} else {
			fmt.Printf("msg success: %s\r\n", m)
		}

	}

}
