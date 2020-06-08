# TestSync
TestSync is a easy use Go based test agent synchronization tool by utitilzing 
action synchronization algorithms and data sharing.

Usable with a simple HTTP API and WebSocket connection.

## Prerequisites
Go - `brew install go`

## Usage
To use this synchronization tool all you have to do is clone this repository
1) `git clone git@github.com:BeyondZeroLV/TestSync.git`
2) Download all Go modules - `go mod download`
3) Build binary - `go build`
4) Create config file
5) Launch executable

## Configuration file example
```
{
    "http_port": 9104,
    "ws_port": 9105,
    
    "logging": {
        "level": "DEBUG"
      },

    "sync_client" : {
      "username": "exampleUserName",
      "password": "examplePassWord"
    }    
}
```