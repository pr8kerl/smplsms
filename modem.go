package main

import (
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

var tpmr int = 0

func init() {
	rand.Seed(time.Now().UnixNano())
	// fmt.Println(createUDH(3))
}

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
	var e error = nil
	m.Port.Flush()
	_, err := m.Port.Write([]byte(command))
	if err != nil {
		log.Printf("error writing to port: %s\r\n", err)
		return status, err
	}
	buf := make([]byte, 32)
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
			if strings.Contains(status, "OK\r\n") {
				break
			} else if strings.Contains(status, "ERR\r\n") {
				e = errors.New(status)
				break
			}
		}
	}
	return status, e
}

func (m *GSMModem) SendSMS(mobile string, message string) (string, error) {
	log.Println("SendSMS ", mobile, message)
	mobile = strings.Replace(mobile, "+", "", -1)
	// detected a double-width char
	if len([]rune(message)) < len(message) {
		log.Println("This is UNICODE sms. Will use PDU mode")
		return m.SendPduSMS(mobile, message)
	}
	// Put Modem in SMS Text Mode
	resp, err := m.SendCommand("AT+CMGF=1\r", false)
	if err != nil {
		log.Println("modem error setting text mode")
		return resp, err
	}

	m.SendCommand("AT+CMGS=\""+mobile+"\"\r", false)
	if err != nil {
		log.Println("modem error setting mobile number")
		return resp, err
	}

	// EOM CTRL-Z = 26
	return m.SendCommand(message+string(26), true)
}

func (m *GSMModem) SendPduSMS(mobile string, message string) (string, error) {
	log.Println("SendPduSMS ", mobile, message)
	if len([]rune(message)) > 70 {
		log.Println("This is long message. Will split")
		return m.SendLongPduSms(mobile, message)
	}
	// Put Modem in SMS Binary Mode
	resp, err := m.SendCommand("AT+CMGF=0\r", false)
	if err != nil {
		log.Println("modem pdu error setting binary mode")
		return resp, err
	}

	telNumber := "01" + "00" + fmt.Sprintf("%02X", len(mobile)) + "91" + encodePhoneNumber(mobile)
	encodedText := encodeUcs2ToString(message)
	textLen := lenInHex(encodedText)
	text := telNumber + "0008" + textLen + encodedText

	resp, err = m.SendCommand("AT+CMGS="+strconv.Itoa(lenInBytes(text))+"\r", false)
	if err != nil {
		log.Println("modem pdu error setting phone number")
		return resp, err
	}
	text = "00" + text
	// EOM CTRL-Z = 26
	resp, err = m.SendCommand(text+string(26), true)
	if err != nil {
		log.Println("modem pdu error sending text")
		return resp, err
	}
	return "", nil
}

func (m *GSMModem) SendLongPduSms(mobile string, message string) (string, error) {
	mes := []rune(message)
	numberOfMessages := len(mes) / 67
	if len(mes)%67 > 0 {
		numberOfMessages++
	}
	log.Println("Total messages", numberOfMessages, "length:", len(message))
	udh := createUDH(numberOfMessages)
	encodedPhoneNumber := encodePhoneNumber(mobile)
	phoneLength := fmt.Sprintf("%02X", len(mobile))
	var resp string
	var err error
	for i := 0; i < numberOfMessages; i++ {
		resp, err = m.SendCommand("AT+CMGF=0\r", false)
		if err != nil {
			log.Println("modem long pdu error setting binary mode")
			return resp, err
		}
		telNumber := "41" + getNextTpmr() + phoneLength + "91" + encodedPhoneNumber
		startByte := i * 67
		stopByte := (i + 1) * 67
		if stopByte >= len(mes) {
			stopByte = len(mes) - 1
		}
		text := string(mes[startByte:stopByte])
		log.Println(startByte, stopByte, text)
		encodedText := encodeUcs2ToString(text)
		textLen := lenInHex(udh[i] + encodedText)
		text = telNumber + "0008" + textLen + udh[i] + encodedText
		resp, err = m.SendCommand("AT+CMGS="+strconv.Itoa(lenInBytes(text))+"\r", false)
		if err != nil {
			log.Println("modem long pdu error setting phone number")
			return resp, err
		}
		text = "00" + text
		resp, err = m.SendCommand(text+string(26), true)
		if err != nil {
			log.Println("modem long pdu error sending text messsage")
			return resp, err
		}
	}
	return "", nil
}

func lenInHex(str string) string {
	return fmt.Sprintf("%02X", lenInBytes(str))
}

func lenInBytes(str string) int {
	return int(float64(len(str))/2 + 0.9999)
}

func encodePhoneNumber(phone string) string {
	if (len(phone) % 2) != 0 {
		phone += "F"

	}
	str := []rune(phone)
	for i := 0; i < len(str); i += 2 {
		str[i], str[i+1] = str[i+1], str[i]
	}
	return string(str)
}

func createUDH(slices int) []string {
	result := make([]string, slices)
	IED1 := fmt.Sprintf("%02X", rand.Intn(255))
	base := "05" + "00" + "03" + IED1 + fmt.Sprintf("%02X", slices)
	for i := 0; i < slices; i++ {
		result[i] = base + fmt.Sprintf("%02X", i+1)
	}
	return result
}

func getNextTpmr() string {
	tpmr++
	if tpmr == 256 {
		tpmr = 0
	}
	return fmt.Sprintf("%02X", tpmr)
}

// pdu code

// ErrUnevenNumber happens when the number of octets (bytes) in the input is uneven.
var ErrUnevenNumber = errors.New("decode ucs2: uneven number of octets")

// EncodeUcs2 encodes the given UTF-8 text into UCS2 (UTF-16) encoding and returns the produced octets.
func encodeUcs2(str string) []byte {
	buf := utf16.Encode([]rune(str))
	octets := make([]byte, 0, len(buf)*2)
	for _, n := range buf {
		octets = append(octets, byte(n&0xFF00>>8), byte(n&0x00FF))
	}
	return octets
}

// EncodeUcs2 encodes the given UTF-8 text into UCS2 (UTF-16) encoding and returns the produced string.
func encodeUcs2ToString(str string) string {
	return fmt.Sprintf("%02X", encodeUcs2(str))
}

// DecodeUcs2 decodes the given UCS2 (UTF-16) octet data into a UTF-8 encoded string.
func decodeUcs2(octets []byte) (str string, err error) {
	if len(octets)%2 != 0 {
		err = ErrUnevenNumber
		return
	}
	buf := make([]uint16, 0, len(octets)/2)
	for i := 0; i < len(octets); i += 2 {
		buf = append(buf, uint16(octets[i])<<8|uint16(octets[i+1]))
	}
	runes := utf16.Decode(buf)
	return string(runes), nil
}
