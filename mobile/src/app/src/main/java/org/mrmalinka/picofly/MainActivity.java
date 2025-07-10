package org.mrmalinka.picofly;

import android.annotation.SuppressLint;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.hardware.usb.UsbDevice;
import android.hardware.usb.UsbDeviceConnection;
import android.hardware.usb.UsbManager;
import android.os.Build;
import android.os.Bundle;
import android.util.Log;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;
import androidx.appcompat.app.AppCompatActivity;
import com.hoho.android.usbserial.driver.CdcAcmSerialDriver;
import com.hoho.android.usbserial.driver.ProbeTable;
import com.hoho.android.usbserial.driver.UsbSerialDriver;
import com.hoho.android.usbserial.driver.UsbSerialPort;
import com.hoho.android.usbserial.driver.UsbSerialProber;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class MainActivity extends AppCompatActivity {
    private static final String TAG = "Picofly";
    private static final String ACTION_USB_PERMISSION = "org.mrmalinka.picofly.USB_PERMISSION";
    private static final int PICO_VENDOR_ID = 0x2E8A; // pico VID
    private static final int PICO_PRODUCT_ID = 0x0005; // common pico PID

    private UsbSerialPort serialPort;
    private UsbManager usbManager;
    private final ExecutorService serialExecutor = Executors.newSingleThreadExecutor();

    private final BroadcastReceiver usbReceiver = new BroadcastReceiver() {
        @Override
        public void onReceive(Context context, Intent intent) {
            String action = intent.getAction();
            if (UsbManager.ACTION_USB_DEVICE_ATTACHED.equals(action)) {
                UsbDevice device = intent.getParcelableExtra(UsbManager.EXTRA_DEVICE);
                if (device != null && isPicoDevice(device)) {
                    connectToDevice(device);
                }
            } else if (UsbManager.ACTION_USB_DEVICE_DETACHED.equals(action)) {
                UsbDevice device = intent.getParcelableExtra(UsbManager.EXTRA_DEVICE);
                if (device != null && isPicoDevice(device)) {
                    closeSerialPort();
                }
            } else if (ACTION_USB_PERMISSION.equals(action)) {
                UsbDevice device = intent.getParcelableExtra(UsbManager.EXTRA_DEVICE);
                if (intent.getBooleanExtra(UsbManager.EXTRA_PERMISSION_GRANTED, false)) {
                    if (device != null) {
                        connectToDevice(device);
                    }
                }
            }
        }
    };

    @SuppressLint("SetJavaScriptEnabled")
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        usbManager = (UsbManager) getSystemService(Context.USB_SERVICE);

        WebView webView = findViewById(R.id.webview);
        WebSettings webSettings = webView.getSettings();
        webSettings.setJavaScriptEnabled(true);
        webSettings.setDomStorageEnabled(true);
        webView.addJavascriptInterface(new WebAppInterface(), "AndroidSerial");
        webView.loadUrl("file:///android_asset/index.html");

        IntentFilter filter = new IntentFilter();
        filter.addAction(UsbManager.ACTION_USB_DEVICE_ATTACHED);
        filter.addAction(UsbManager.ACTION_USB_DEVICE_DETACHED);
        filter.addAction(ACTION_USB_PERMISSION);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(usbReceiver, filter, Context.RECEIVER_NOT_EXPORTED);
        } else {
            throw new RuntimeException("App will only run on Android 12 or greater!");
        }

        checkConnectedDevices();
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        unregisterReceiver(usbReceiver);
        closeSerialPort();
        serialExecutor.shutdownNow();
    }

    private void checkConnectedDevices() {
        HashMap<String, UsbDevice> deviceList = usbManager.getDeviceList();
        for (UsbDevice device : deviceList.values()) {
            if (isPicoDevice(device)) {
                requestUsbPermission(device);
                break;
            }
        }
    }

    private boolean isPicoDevice(UsbDevice device) {
        return device.getVendorId() == PICO_VENDOR_ID &&
                (device.getProductId() == PICO_PRODUCT_ID ||
                        device.getProductId() == 0x000A); // other pico PID
    }

    private void connectToDevice(UsbDevice device) {
        serialExecutor.execute(() -> {
            try {
                if (serialPort != null) {
                    try {
                        serialPort.close();
                    } catch (IOException e) {
                        Log.e(TAG, "Error closing previous connection", e);
                    }
                }

                ProbeTable customTable = new ProbeTable();
                customTable.addProduct(PICO_VENDOR_ID, PICO_PRODUCT_ID, CdcAcmSerialDriver.class);
                customTable.addProduct(PICO_VENDOR_ID, 0x000A, CdcAcmSerialDriver.class);

                UsbSerialProber prober = new UsbSerialProber(customTable);
                UsbSerialDriver driver = prober.probeDevice(device);

                if (driver == null) {
                    Log.w(TAG, "No compatible driver found for device");
                    return;
                }

                if (driver.getPorts().isEmpty()) {
                    Log.w(TAG, "No serial ports available on device");
                    return;
                }

                UsbDeviceConnection connection = usbManager.openDevice(device);
                if (connection == null) {
                    requestUsbPermission(device);
                    return;
                }

                serialPort = driver.getPorts().get(0);
                serialPort.open(connection);
                serialPort.setParameters(115200, 8, UsbSerialPort.STOPBITS_1, UsbSerialPort.PARITY_NONE);
                if (serialPort != null) {
                    Log.i(TAG, "Connected to: " + device.getDeviceName());
                }
                Log.i(TAG, "USB connection established");

            } catch (Exception e) {
                Log.e(TAG, "Error connecting to USB device", e);
            }
        });
    }

    private void requestUsbPermission(UsbDevice device) {
        PendingIntent permissionIntent = PendingIntent.getBroadcast(
                this,
                0,
                new Intent(ACTION_USB_PERMISSION),
                PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT
        );

        if (usbManager.hasPermission(device)) {
            connectToDevice(device);
        } else {
            usbManager.requestPermission(device, permissionIntent);
        }
    }

    private void closeSerialPort() {
        serialExecutor.execute(() -> {
            if (serialPort != null) {
                try {
                    serialPort.close();
                    serialPort = null;
                    Log.i(TAG, "USB connection closed");
                } catch (IOException e) {
                    Log.e(TAG, "Error closing USB connection", e);
                }
            }
        });
    }

    public class WebAppInterface {
        @JavascriptInterface
        public boolean isConnected() {
            return serialPort != null;
        }

        @JavascriptInterface
        public void writeUsb(String data) {
            if (serialPort == null) return;

            serialExecutor.execute(() -> {
                try {
                    byte[] buffer = data.getBytes(StandardCharsets.UTF_8);
                    serialPort.write(buffer, 1000);
                    Log.d(TAG, "Sent: " + data);
                } catch (IOException e) {
                    Log.e(TAG, "USB write error", e);
                    closeSerialPort();
                }
            });
        }

        @JavascriptInterface
        public String readUsb(int bytesToRead) {
            if (serialPort == null) return "";

            try {
                byte[] buffer = new byte[bytesToRead];
                int bytesRead = serialPort.read(buffer, 1000);
                if (bytesRead > 0) {
                    return new String(buffer, 0, bytesRead, StandardCharsets.UTF_8);
                }
            } catch (IOException e) {
                Log.e(TAG, "USB read error", e);
                closeSerialPort();
            }
            return "";
        }
    }
}