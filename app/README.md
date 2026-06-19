# ergonomos client (Flutter)

The Flutter client for web, iOS and Android. Not scaffolded yet — it is added
once the first server endpoints exist and the OpenAPI-generated Dart client is
wired in.

## Flutter version

The version is pinned with [fvm](https://fvm.app) in `.fvmrc`. It currently
tracks `stable`; **pin an exact version** for reproducible builds once you
choose one:

```sh
brew install fvm           # or: dart pub global activate fvm
cd app
fvm releases               # list available versions
fvm use 3.x.y              # pins an exact version in .fvmrc
fvm install
```

Then run Flutter through fvm, e.g. `fvm flutter create .`, `fvm flutter run`.
