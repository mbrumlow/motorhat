package motorhat

import (
	"fmt"
	"math"
	"time"

	"github.com/mbrumlow/i2c"
)

const (
	mode1      = 0x00
	mode2      = 0x01
	preescale  = 0xFE
	led0OnL    = 0x06
	led0OnH    = 0x07
	led0OffL   = 0x08
	led0OffH   = 0x09
	allLedOnL  = 0xFA
	allLedOnH  = 0xFB
	allLedOffL = 0xFC
	allLedOffH = 0xFD

	sleep   = 0x10
	allcall = 0x01
	outdrv  = 0x04
)

var (
	motorMapPWM = map[int]int{
		1: 8,
		2: 13,
		3: 2,
		4: 7,
	}
	motorMapIn1 = map[int]int{
		1: 10,
		2: 11,
		3: 4,
		4: 5,
	}
	motorMapIn2 = map[int]int{
		1: 9,
		2: 12,
		3: 3,
		4: 6,
	}
)

type MotorHat struct {
	i2c *i2c.I2C
}

func Open(addr uint8, bus int) (*MotorHat, error) {

	i2c, err := i2c.New(addr, bus)
	if err != nil {
		return nil, err
	}

	mh := &MotorHat{i2c: i2c}

	if err := mh.init(); err != nil {
		mh.Close()
		return nil, err
	}

	return mh, nil
}

func (mh *MotorHat) init() error {

	if err := mh.setAllPWM(0, 0); err != nil {
		return err
	}

	if err := mh.initPWM(); err != nil {
		return err
	}

	if err := mh.setPWMFreq(1600); err != nil {
		return err
	}
	return nil
}

func (mh *MotorHat) Close() {
	mh.i2c.Close()
}

func (mh *MotorHat) Speed(m, s int) error {

	pwm, _, _, err := getMotor(m)
	if err != nil {
		return err
	}

	if s < 0 {
		s = 0
	} else if s > 255 {
		s = 255
	}

	mh.setPWM(pwm, 0, s*16)

	return nil
}

func (mh *MotorHat) Forward(m int) error {

	_, in1, in2, err := getMotor(m)
	if err != nil {
		return err
	}

	mh.setPin(in1, 1)
	mh.setPin(in2, 0)

	return nil
}

func (mh *MotorHat) Backward(m int) error {

	_, in1, in2, err := getMotor(m)
	if err != nil {
		return err
	}

	mh.setPin(in1, 0)
	mh.setPin(in2, 1)

	return nil
}

func (mh *MotorHat) Stop(m int) error {

	_, in1, in2, err := getMotor(m)
	if err != nil {
		return err
	}

	mh.setPin(in1, 0)
	mh.setPin(in2, 0)

	return nil
}

func (mh *MotorHat) setPin(pin, value int) error {

	if value == 1 {
		mh.setPWM(pin, 4096, 0)
	} else if value == 0 {
		mh.setPWM(pin, 0, 4096)
	}

	return nil
}

func (mh *MotorHat) initPWM() error {

	if err := mh.i2c.WriteRegister(mode2, outdrv); err != nil {
		return err
	}

	if err := mh.i2c.WriteRegister(mode1, allcall); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	m, err := mh.i2c.ReadRegister(mode1)
	if err != nil {
		return err
	}

	m = m &^ sleep
	if err := mh.i2c.WriteRegister(mode1, m); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	return nil
}

func (mh *MotorHat) setPWMFreq(freq int) error {

	ps := 25000000.0
	ps /= 4096.0
	ps /= float64(freq)
	ps -= 1.0
	ps = math.Floor(ps + 0.05)

	oldmode, err := mh.i2c.ReadRegister(mode1)
	if err != nil {
		return err
	}

	writeReg := func(r, v uint8) {
		if err != nil {
			return
		}

		err = mh.i2c.WriteRegister(r, v)
	}

	newmode := (oldmode & 0x7F) | sleep

	writeReg(mode1, newmode)

	writeReg(preescale, uint8(math.Floor(ps)))
	writeReg(mode1, oldmode)

	time.Sleep(5 * time.Millisecond)

	writeReg(mode1, oldmode|0x80)

	return err
}

func (mh *MotorHat) setPWM(pin, on, off int) error {

	var err error

	writeReg := func(r, v uint8) {
		if err != nil {
			return
		}

		err = mh.i2c.WriteRegister(r, v)
	}

	writeReg(led0OnL+uint8(4*pin), uint8(on&0xFF))
	writeReg(led0OnH+uint8(4*pin), uint8(on>>8))

	writeReg(led0OffL+uint8(4*pin), uint8(off&0xFF))
	writeReg(led0OffH+uint8(4*pin), uint8(off>>8))

	return err
}

func (mh *MotorHat) setAllPWM(on, off int) error {

	var err error

	writeReg := func(r, v uint8) {
		if err != nil {
			return
		}

		err = mh.i2c.WriteRegister(r, v)
	}

	writeReg(allLedOnL, uint8(on&0xFF))
	writeReg(allLedOnH, uint8(on>>8))

	writeReg(allLedOffL, uint8(off&0xFF))
	writeReg(allLedOffH, uint8(off>>8))

	return err
}

func getMotor(m int) (int, int, int, error) {

	pwm, ok := motorMapPWM[m]
	if !ok {
		return 0, 0, 0, fmt.Errorf("Motor map not found!")
	}

	in1, ok := motorMapIn1[m]
	if !ok {
		return 0, 0, 0, fmt.Errorf("Motor map not found!")
	}

	in2, ok := motorMapIn2[m]
	if !ok {
		return 0, 0, 0, fmt.Errorf("Motor map not found!")
	}

	return pwm, in1, in2, nil
}
