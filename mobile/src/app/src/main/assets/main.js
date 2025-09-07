// -------
// helpers
// -------

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function base64Decode(base64String) {
  if (!base64String || base64String === "") {
    return new Uint8Array(0);
  }

  try {
    let binaryString = atob(base64String);

    let bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }

    return bytes;
  } catch (error) {
    Android.internalLogJS("Base64 decode err:", error);
    return new Uint8Array(0);
  }
}

// -------
// status
// -------

const status_none = 0;
const status_idle = 1;
const status_readyForTakeoff = 2;
const status_flying = 3;
const status_circling = 4;
const status_landing = 5;

const statusMap = {
  [status_none]: "None",
  [status_idle]: "Idle",
  [status_readyForTakeoff]: "Ready",
  [status_flying]: "Flying",
  [status_circling]: "Circling",
  [status_landing]: "Landing",
};

function percentageToUint32(f) {
  const clamped = Math.min(Math.max(f, 0), 100);
  return Math.floor((clamped / 100) * 0xffffffff);
}

function percentageFromUint32(u) {
  return (u / 0xffffffff) * 100;
}

function planeStatusFromBytes(data) {
  if (!(data instanceof Uint8Array) || data.byteLength !== 29) {
    return null;
  }
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);

  return {
    status: view.getUint8(0),
    battery: percentageFromUint32(view.getUint32(1, false)),
    speed: view.getFloat32(5, false),
    altitude: view.getFloat32(9, false),
    latitude: view.getFloat64(13, false),
    longitude: view.getFloat64(21, false),
  };
}

// -------
// protocol
// -------

const payloadType_error = 0;
const payloadType_bulk = 1;
const payloadType_rssi = 2;
const payloadType_wpSet = 3;
const payloadType_altSet = 4;
const payloadType_takeoff = 5;
const payloadType_land = 6;
// for manual control only
const payloadType_joystick = 7;
const payloadType_throttle = 8;
const payloadType_errorInternal = 0xff;

function newPacket(payloadType, payload) {
  // these packets are meant for everything from
  // lora to usb and as such do not have to be modified when forwarded
  // packet structure:
  //  header - 2 bytes
  //    first - length of the full packet including header
  //    second - data type of payload
  //
  //  payload - n bytes
  return [payload.length + 2, payloadType, ...payload];
}

function parsePacket(packet) {
  if (packet.length === 0) {
    return {
      payloadType: payloadType_errorInternal,
      payload: null,
    };
  }

  //const packetLength = packet[0];
  const payloadType = packet[1];
  const payload = packet.slice(2);

  return {
    payloadType: payloadType,
    payload: payload,
  };
}

// -------
// main
// -------

function buttonRelocate() {
  if (!Alpine.store("map").map) {
    Android.internalLogJS("Map not initialized");
    return;
  }

  const center = Alpine.store("map").map.getCenter();
  const buffer = new ArrayBuffer(16);
  const view = new DataView(buffer);
  view.setFloat64(0, center.lat, false);
  view.setFloat64(8, center.lng, false);

  const payload = Array.from(new Uint8Array(buffer));
  Android.usbWrite(newPacket(payloadType_wpSet, payload));
}

window.updateUsbStatusText = function (text) {
  Alpine.store("connections").usb = text;
};

window.onNewData = function (base64) {
  if (base64.startsWith("!")) {
    Android.internalLogJS("Empty");
    return;
  }

  let result = parsePacket(base64Decode(base64));

  if (result.payloadType == payloadType_errorInternal) {
    Android.internalLogJS("Parse error");
  } else {
    switch (result.payloadType) {
      case payloadType_error:
        Android.internalLogJS("External error: " + result.payload);
      case payloadType_bulk:
        planeStatus = planeStatusFromBytes(result.payload);
        if (planeStatus == null) {
          Android.internalLogJS("Malformed compressed status");
        }

        Alpine.store("telemetry").Status = statusMap[planeStatus.status];
        Alpine.store("telemetry").Battery =
          planeStatus.battery.toFixed(1).toString() + "%";
        Alpine.store("telemetry").Speed =
          planeStatus.speed.toFixed(2).toString() + "m/s";
        Alpine.store("telemetry").Altitude =
          planeStatus.altitude.toFixed(2).toString() + "m";

        Alpine.store("map").map.panTo([
          planeStatus.latitude,
          planeStatus.longitude,
        ]);
        break;
      case payloadType_rssi:
        const dataView = new DataView(result.payload.buffer);
        const rssi = dataView.getInt32(0, false);
        Alpine.store("connections").lora = rssi.toFixed(0) + "dBm";
        break;
      default:
        Android.internalLogJS("Invalid packet");
    }
  }
};

const alt = "..."; // default string displayed before stuff loads in
document.addEventListener("alpine:init", () => {
  Alpine.store("connections", {
    usb: alt,
    lora: alt,
  });

  Alpine.store("telemetry", {
    Status: alt,
    Battery: alt,
    Speed: alt,
    Altitude: alt,
  });

  Alpine.store("camera", {
    camStatus: alt,
  });

  Alpine.store("map", {
    map: null,
  });
});

if (typeof window.Android !== "object") {
  window.Android = {
    loadAssetToWebView(path) {
      window.location.href = "/" + path;
    },
  };
}
