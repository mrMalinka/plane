from i2c_lcd import I2cLcd

import usb_cdc, busio, board, time, digitalio

def simple_number_hash(input_val: int) -> bytes:
    x = input_val & 0xFFFFFFFF
    h = ((x ^ 0x00ABCDEF) * 2654435761) & 0xFFFFFFFF
    h = ((h >> 16) ^ h) & 0xFFFFFFFF
    return h.to_bytes(4, "little")

data_serial = usb_cdc.data
lcd = I2cLcd(busio.I2C(scl=board.GP1, sda=board.GP0), 0x27, 2, 16)
onboard_led = digitalio.DigitalInOut(board.LED)
onboard_led.direction = digitalio.Direction.OUTPUT
onboard_led.value = False

# wait until usb is on and blink to indicate it isnt
while not data_serial.connected:
    onboard_led.value = True
    time.sleep(0.4)
    onboard_led.value = False
    time.sleep(0.4)

lcd.clear()
lcd.putstr("waiting")
# expecting a conntest
while True:
    buf = data_serial.read(4)
    if buf is None or len(buf) < 4:
        continue
    num = int.from_bytes(buf, "little")

    lcd.clear()
    lcd.putstr("read: " + str(num))

    data_serial.write(simple_number_hash(num))
    lcd.move_to(0, 1)
    lcd.putstr("sent reply")