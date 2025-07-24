package org.mrmalinka.picofly;

import android.annotation.SuppressLint;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.hardware.usb.UsbDevice;
import android.hardware.usb.UsbManager;
import android.os.Bundle;
import android.util.Log;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Toast;
import android.hardware.usb.UsbDeviceConnection;
import android.hardware.usb.UsbInterface;
import android.hardware.usb.UsbEndpoint;
import android.hardware.usb.UsbConstants;

import com.hoho.android.usbserial.driver.UsbSerialDriver;
import com.hoho.android.usbserial.driver.UsbSerialPort;

import androidx.appcompat.app.AppCompatActivity;

import org.json.JSONObject;

public class MainActivity extends AppCompatActivity {
    private enum UsbPermission { Unknown, Requested, Granted, Denied }

    private static final String TAG = "PicoFly";
    private static final String ACTION_USB_PERMISSION = "org.mrmalinka.picofly.USB_PERMISSION";
    private static final boolean ENABLE_DEBUG_LOGGING = true;

    private final BroadcastReceiver broadcastReceiver;
    private WebView webView;

    private UsbSerialPort usbSerialPort;
    private UsbPermission usbPermission = UsbPermission.Unknown;

    public MainActivity() {
        broadcastReceiver = new BroadcastReceiver() {
            @Override
            public void onReceive(Context context, Intent intent) {
                String action = intent.getAction();
                internalLog("Broadcast: " + action);

                if (UsbManager.ACTION_USB_DEVICE_ATTACHED.equals(action)) {
                    internalLog("Attached, connecting");
                    try {
                        connect();
                    } catch (Exception e) {
                        internalLog("Connect exception: " + e);
                    }
                }
                if (ACTION_USB_PERMISSION.equals(action)) {
                    usbPermission = intent.getBooleanExtra(UsbManager.EXTRA_PERMISSION_GRANTED, false)
                            ? UsbPermission.Granted : UsbPermission.Denied;
                    if (usbPermission == UsbPermission.Granted)
                        internalLog("Granted");
                    else
                        internalLog("Denied");
                    connect();
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
            internalLog("Device not found");
            return;
        }

        // get driver
        UsbSerialDriver driver = CustomProber.getCustomProber().probeDevice(device);
        if (driver == null) {
            internalLog("No driver");
            return;
        }


        internalLog(driver.getPorts().size() + " ports");
        usbSerialPort = driver.getPorts().get(0);
        UsbDeviceConnection usbConnection = usbManager.openDevice(driver.getDevice());

        if (usbConnection == null && usbPermission == UsbPermission.Unknown && !usbManager.hasPermission(driver.getDevice())) {
            // ask for permission
            usbPermission = UsbPermission.Requested;

            Intent intent = new Intent(ACTION_USB_PERMISSION);
            intent.setPackage(getPackageName());
            PendingIntent usbPermissionIntent = PendingIntent.getBroadcast(
                   this,
                   0,
                    intent,
                    PendingIntent.FLAG_MUTABLE
            );

            internalLog("Permission requested");
            usbManager.requestPermission(driver.getDevice(), usbPermissionIntent);
            return;
        }
        if (usbConnection == null) {
            if (!usbManager.hasPermission(driver.getDevice()))
                internalLog("No permission & connection");
            else
                internalLog("No connection");
            return;
        }

        StringBuilder logHist = new StringBuilder();
        for (UsbSerialPort port : driver.getPorts()) {
            try {
                port.open(usbConnection);
                port.setParameters(115200, 8, 1, UsbSerialPort.PARITY_NONE);
                usbSerialPort = port;
                internalLog("Connected to port");
                return;
            } catch (Exception e) {
                logHist.append("|Fail: ").append(e);
                try { if (port.isOpen()) port.close(); } catch (Exception ignored) {}
            }
        }

        usbConnection.close();
        internalLog(logHist.toString());
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setupUsb();
        setupWebView();
        webView.loadUrl("file:///android_asset/index.html");

        internalLog("onCreate completed");
    }
    private void setupUsb() {
        internalLog("Setting up usb");

        IntentFilter filter = new IntentFilter();
        filter.addAction(UsbManager.ACTION_USB_DEVICE_ATTACHED);
        filter.addAction(ACTION_USB_PERMISSION);
        registerReceiver(broadcastReceiver, filter, Context.RECEIVER_NOT_EXPORTED);
    }
    @SuppressLint("SetJavaScriptEnabled")
    private void setupWebView() {
        internalLog("Setting up WebView");

        webView = new WebView(this);
        setContentView(webView);

        WebSettings webSettings = webView.getSettings();
        webSettings.setJavaScriptEnabled(true);
        webSettings.setDomStorageEnabled(true);
        webSettings.setAllowFileAccess(true);
        webSettings.setAllowContentAccess(true);

        webView.addJavascriptInterface(new AndroidInterface(), "Android");

        internalLog("WebView setup completed");
    }

    private void internalLog(String message) {
        if (ENABLE_DEBUG_LOGGING) {
            Log.d(TAG, message);
            runOnUiThread(() -> Toast.makeText(this, message, Toast.LENGTH_SHORT).show());
            if (webView != null)
                runOnUiThread(() -> webView.evaluateJavascript(
                       "javascript:window.updateUsbStatusText(" +  JSONObject.quote(message) + ")",
                       null
                ));
        }
    }

    public class AndroidInterface {
        @JavascriptInterface
        public void loadAssetToWebView(String path) {
            internalLog("Loading asset: " + path);
            runOnUiThread(() -> webView.loadUrl("file:///android_asset/" + path));
        }

        @JavascriptInterface
        public void internalLogJS(String message) {
            internalLog(message);
        }
    }
}
