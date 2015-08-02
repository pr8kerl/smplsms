package main

import (
	"errors"
	"github.com/tarm/serial"
	"log"
	"strings"
	"time"
)

type GSMModem struct {
	ComPort  string
	BaudRate int
	Port     *serial.Port
	DeviceId string
}

func NewModem(ComPort string, BaudRate int, DeviceId string) (modem *GSMModem) {
	modem = &GSMModem{ComPort: ComPort, BaudRate: BaudRate, DeviceId: DeviceId}
	return modem
}

func (m *GSMModem) Connect() (err error) {
	config := &serial.Config{Name: m.ComPort, Baud: m.BaudRate, ReadTimeout: time.Second}
	m.Port, err = serial.OpenPort(config)
	return err
}

func (m *GSMModem) SendCommand(command string, waitForOk bool) error {
	log.Println("--- SendCommand: ", command)
	var status string = ""
	var e error
	m.Port.Flush()
	_, err := m.Port.Write([]byte(command))
	if err != nil {
		log.Printf("error writing to port: %s", err)
		return err
	}
	buf := make([]byte, 64)
	var loop int = 1
	if waitForOk {
		loop = 10
	}
	for i := 0; i < loop; i++ {
		// ignoring error as EOF raises error on Linux
		n, _ := m.Port.Read(buf)
		if n > 0 {
			status = string(buf[:n])
			log.Printf("SendCommand: rcvd %d bytes: %s\r\n", n, status)
			if strings.HasPrefix(status, "OK") {
				break
			} else if strings.HasPrefix(status, "ERR") {
				e = errors.New(status)
				break
			}
		}
	}
	return e
}

func (m *GSMModem) SendSMS(mobile string, message string) error {
	log.Println("--- SendSMS ", mobile, message)

	// Put Modem in SMS Text Mode
	m.SendCommand("AT+CMGF=1\r\n", false)

	m.SendCommand("AT+CMGS=\""+mobile+"\"\r\n", false)

	// EOM CTRL-Z = 26
	return m.SendCommand(message+string(26), true)

}
