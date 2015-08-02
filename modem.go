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
	cfg := &serial.Config{Name: m.ComPort, Baud: m.BaudRate, ReadTimeout: time.Second}
	m.Port, err = serial.OpenPort(cfg)
	return err
}

func (m *GSMModem) InitialiseModem() (err error) {

	if len(config.ModemInitString) > 0 {
		log.Printf("sending init string to modem: %s\r\n", config.ModemInitString)
		resp, err := m.SendCommand(config.ModemInitString+"\x13", false)
		if err != nil {
			log.Println("modem init response: ", resp)
			return err
		}
	}
	// Put Modem in SMS Text Mode
	resp, err := m.SendCommand("AT+CMGF=1\r", false)
	if err != nil {
		log.Printf("modem response: %s\r\n", resp)
		return err
	}
	if config.Debug {
		resp, err := m.SendCommand("AT!GSMINFO\x13", false)
		if err != nil {
			log.Printf("modem response: %s\r\n", resp)
			return err
		}
		log.Println("GSMINFO 2G network info: ", resp)
		resp, err = m.SendCommand("AT!GSTATUS\x13", false)
		if err != nil {
			log.Printf("modem response: %s\r\n", resp)
			return err
		}
		log.Println("GSTATUS operational status: ", resp)
		resp, err = m.SendCommand("AT!GVER\x13", false)
		if err != nil {
			log.Printf("modem response: %s\r\n", resp)
			return err
		}
		log.Printf("GVER firmware version: %s\r\n", resp)
	}

	return nil
}

func (m *GSMModem) SendCommand(command string, waitForOk bool) (string, error) {
	log.Printf("SendCommand: %s\r\n", command)
	var status string = ""
	var e error
	m.Port.Flush()
	_, err := m.Port.Write([]byte(command))
	if err != nil {
		log.Printf("error writing to port: %s\r\n", err)
		return status, err
	}
	buf := make([]byte, 64)
	var loop int = 1
	if waitForOk {
		loop = 24
	}
	for i := 0; i < loop; i++ {
		// ignoring error as EOF raises error on Linux
		n, _ := m.Port.Read(buf)
		if n > 0 {
			status = string(buf[:n])
			log.Printf("SendCommand: rcvd %d bytes: %s\r\n", n, status)
			if strings.HasSuffix(status, "OK") {
				break
			} else if strings.HasSuffix(status, "ERR") {
				e = errors.New(status)
				break
			}
		}
	}
	return string(buf), e
}

func (m *GSMModem) SendSMS(mobile string, message string) error {

	log.Printf("SendSMS %s %s\r\n", mobile, message)

	resp, err := m.SendCommand("AT+CMGS=\""+mobile+"\"\r", false)
	if err != nil {
		log.Println("SendSMS error response:", resp)
		return err
	}

	// EOM CTRL-Z = 26
	// return m.SendCommand(message+string(26), true)
	resp, err = m.SendCommand(message+string(26), false)
	if err != nil {
		log.Printf("SendSMS error response: %s\r\n", resp)
		return err
	}

	return nil

}
