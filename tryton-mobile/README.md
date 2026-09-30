# Tryton Mobile (Flutter)

Mobile client for Talon targeting iOS and Android.

## Prerequisites

- Flutter 3.32+ (Dart 3.4+)

## Run

```bash
flutter pub get
flutter run --dart-define=TALON_URL=http://10.0.2.2:8000
```

`10.0.2.2` is the Android emulator alias for the host machine. Use the LAN
address of the Talon server on physical devices.

## Features in this scaffold

- Database selection and session login (`common.db.login` + session cookie)
- Menu and model browser
- Tree/list view with search, pull-to-refresh and modal form editing
- JSON-RPC client with tagged value codecs (`lib/codec.dart`)
- Notification bus long-poll client (`lib/bus.dart`)

## Packaging

```bash
flutter build apk --release
flutter build appbundle --release
flutter build ipa --release
```

Native extras listed in the roadmap (biometric login, camera/barcode capture,
push notifications, offline SQLite cache) are layered on top of the same
`RpcClient` and `BusClient`.
