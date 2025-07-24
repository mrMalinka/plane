package org.mrmalinka.picofly;

import com.hoho.android.usbserial.driver.CdcAcmSerialDriver;
import com.hoho.android.usbserial.driver.ProbeTable;
import com.hoho.android.usbserial.driver.UsbSerialProber;

class CustomProber {
    static UsbSerialProber getCustomProber() {
        ProbeTable customTable = new ProbeTable();
        customTable.addProduct(0x2E8A, 0x0005, CdcAcmSerialDriver.class);
        customTable.addProduct(0x2E8A, 0x000A, CdcAcmSerialDriver.class);
        return new UsbSerialProber(customTable);
    }
}
