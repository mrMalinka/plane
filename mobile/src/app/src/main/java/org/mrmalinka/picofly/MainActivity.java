package org.mrmalinka.picofly;

import static com.hoho.android.usbserial.driver.UsbSerialProber.getDefaultProber;

import android.annotation.SuppressLint;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.hardware.usb.UsbDevice;
import android.hardware.usb.UsbDeviceConnection;
import android.hardware.usb.UsbManager;
import android.os.Bundle;
import android.util.Log;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;

import com.hoho.android.usbserial.driver.UsbSerialDriver;
import com.hoho.android.usbserial.driver.UsbSerialPort;

import org.json.JSONObject;

import java.io.IOException;
import java.util.Arrays;
import java.util.Base64;

public class MainActivity extends AppCompatActivity {
    private static final String TAG = "PicoFly";
    private static final String ACTION_USB_PERMISSION = "org.mrmalinka.picofly.USB_PERMISSION";
    private static final boolean ENABLE_DEBUG_LOGGING = false;
    private static final int MAX_PACKET_SIZE = 256; // bytes
    private static final int READ_TIMEOUT_MS = 500;
    private static final int WRITE_TIMEOUT_MS = 200;
    private final BroadcastReceiver broadcastReceiver;
    private WebView webView;
    private UsbSerialPort usbSerialPort;
    private boolean destroyed = false; // for the read thread only

    public MainActivity() {
        broadcastReceiver = new BroadcastReceiver() {
            @Override
            public void onReceive(Context context, Intent intent) {
                String action = intent.getAction();
                internalLog("Broadcast: " + action, false);

                if (UsbManager.ACTION_USB_DEVICE_ATTACHED.equals(action)) {
                    setUsbStatus("Attached");
                    try {
                        connect();
                    } catch (Exception e) {
                        internalLog("Connect exception: " + e, true);
                        setUsbStatus(e.toString());
                    }
                }
                if (UsbManager.ACTION_USB_DEVICE_DETACHED.equals(action)) {
                    internalLog("Detached", true);
                    setUsbStatus("Detached");
                    disconnect();
                }
                if (ACTION_USB_PERMISSION.equals(action)) {
                    if (intent.getBooleanExtra(UsbManager.EXTRA_PERMISSION_GRANTED, false))
                        internalLog("Granted", true);
                    else
                        internalLog("Denied", true);
                    try {
                        setUsbStatus("Trying");
                        connect();
                    } catch (Exception e) {
                        internalLog("Connect exception: " + e, true);
                        setUsbStatus(e.toString());
                    }
                }
            }
        };
    }

    private void connect() {
        UsbDevice device = null;
        UsbManager usbManager = (UsbManager) getSystemService(Context.USB_SERVICE);
        for (UsbDevice v : usbManager.getDeviceList().values()) {
            if (v.getVendorId() == 0x2E8A) {
                device = v;
            }
        }
        if (device == null) {
            internalLog("Device not found", true);
            return;
        }

        // get driver
        UsbSerialDriver driver = getDefaultProber().probeDevice(device);
        if (driver == null) {
            internalLog("No driver", true);
            return;
        }
        if (!usbManager.hasPermission(driver.getDevice())) {
            // ask for permission
            Intent intent = new Intent(ACTION_USB_PERMISSION);
            intent.setPackage(getPackageName());
            PendingIntent usbPermissionIntent = PendingIntent.getBroadcast(
                    this,
                    0,
                    intent,
                    PendingIntent.FLAG_MUTABLE
            );

            internalLog("Permission requested", false);
            usbManager.requestPermission(driver.getDevice(), usbPermissionIntent);
            return;
        }

        internalLog(driver.getPorts().size() + " ports", false);
        usbSerialPort = driver.getPorts().get(0);
        UsbDeviceConnection usbConnection = usbManager.openDevice(driver.getDevice());

        if (usbConnection == null) {
            if (!usbManager.hasPermission(driver.getDevice()))
                internalLog("No permission & connection", true);
            else
                internalLog("No connection", true);
            return;
        }

        try {
            usbSerialPort.open(usbConnection);
            usbSerialPort.setDTR(true);
            usbSerialPort.setRTS(true);
            usbSerialPort.setParameters(115200, 8, 1, UsbSerialPort.PARITY_NONE);

            internalLog("Connected to port", false);
            setUsbStatus("Active");
        } catch (Exception e) {
            internalLog("Failed to open port: " + e, true);
            try {
                if (usbSerialPort.isOpen()) usbSerialPort.close();
            } catch (Exception ignored) {
            }
        }
    }

    private void disconnect() {
        try {
            if (usbSerialPort != null && usbSerialPort.isOpen())
                usbSerialPort.close();
        } catch (IOException e) {
            internalLog("Exception closing port: " + e, true);
        }
        setUsbStatus("Disconnected");
        usbSerialPort = null;
    }

    private void write(byte[] data) {
        if (usbSerialPort == null || !usbSerialPort.isOpen()) {
            internalLog("Not connected", true);
            return;
        }
        try {
            // constant timeout of 1 second
            usbSerialPort.write(data, WRITE_TIMEOUT_MS);
        } catch (Exception e) {
            internalLog("Write exception: " + e, true);
        }
    }

    private byte[] read() throws IOException {
        if (usbSerialPort == null || !usbSerialPort.isOpen()) {
            throw new IOException("Not connected");
        }
        byte[] packet = new byte[MAX_PACKET_SIZE];
        int readLen = usbSerialPort.read(packet, MainActivity.READ_TIMEOUT_MS);
        //internalLog("Read: " + readLen + " | " + usbSerialPort.getDevice().getProductName(), true);
        if (readLen < 0) {
            throw new IOException("Error reading from USB port");
        }
        return Arrays.copyOf(packet, readLen);
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setupUsb();
        setupWebView();
        webView.loadUrl("file:///android_asset/index.html");

        // im aware theres a much more efficient way of doing this with some io manager stuff but idc
        new Thread(() -> {
            while (true) {
                if (destroyed) return;
                if (usbSerialPort == null || !usbSerialPort.isOpen()) {
                    try {
                        Thread.sleep(50);
                    } catch (InterruptedException ignored) {
                    }
                    continue;
                }

                try {
                    byte[] data = read();
                    if (data.length == 0) continue;
                    onNewData(data);
                } catch (IOException e) {
                    internalLog("Read err: " + e, true);
                }
            }
        }).start();

        internalLog("onCreate completed", false);
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        unregisterReceiver(broadcastReceiver);
        destroyed = true;
    }

    private void setupUsb() {
        internalLog("Setting up usb", false);

        IntentFilter filter = new IntentFilter();
        filter.addAction(UsbManager.ACTION_USB_DEVICE_ATTACHED);
        filter.addAction(UsbManager.ACTION_USB_DEVICE_DETACHED);
        filter.addAction(ACTION_USB_PERMISSION);
        registerReceiver(broadcastReceiver, filter, Context.RECEIVER_NOT_EXPORTED);
    }

    @SuppressLint("SetJavaScriptEnabled")
    private void setupWebView() {
        internalLog("Setting up WebView", false);

        webView = new WebView(this);
        setContentView(webView);

        WebSettings webSettings = webView.getSettings();
        webSettings.setJavaScriptEnabled(true);
        webSettings.setDomStorageEnabled(true);
        webSettings.setAllowFileAccess(true);
        webSettings.setAllowContentAccess(true);

        webView.addJavascriptInterface(new AndroidInterface(), "Android");

        internalLog("WebView setup completed", false);
    }

    private void internalLog(String message, boolean important) {
        if (important)
            runOnUiThread(() -> Toast.makeText(this, message, Toast.LENGTH_SHORT).show());
        if (ENABLE_DEBUG_LOGGING)
            Log.d(TAG, message);
    }

    private void onNewData(byte[] data) {
        if (webView == null) return;

        runOnUiThread(() -> webView.evaluateJavascript(
                "javascript:window.onNewData("
                        + JSONObject.quote(Base64.getEncoder().encodeToString(data))
                        + ")",
                null
        ));
    }

    private void setUsbStatus(String status) {
        if (webView != null)
            runOnUiThread(() -> webView.evaluateJavascript(
                    "javascript:window.updateUsbStatusText(" + JSONObject.quote(status) + ")",
                    null
            ));
    }

    public class AndroidInterface {
        @JavascriptInterface
        public void loadAssetToWebView(String path) {
            internalLog("Loading asset: " + path, false);
            runOnUiThread(() -> webView.loadUrl("file:///android_asset/" + path));
        }

        @JavascriptInterface
        public void internalLogJS(String message) {
            internalLog(message, true);
        }

        @JavascriptInterface
        public void usbWrite(String base64) {
            write(Base64.getDecoder().decode(base64));
        }

        @JavascriptInterface
        public boolean isConnected() {
            if (usbSerialPort != null)
                return usbSerialPort.isOpen();
            else return false;
        }
    }
}
