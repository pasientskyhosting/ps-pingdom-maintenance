# PatientSky Pingdom Maintenance

## Description
Keep pingdom maintenance schedule up to date


## Quickstart

### Step 1 - Setup env
You need these environment variables:
- `API_KEY` - Pingdom API Key
- `MAINTENANCE_ID` - Pingdom Maintenance ID to update


### Step 2 - Build docker image and run

`make all` to build binaries and create the docker image

`make docker-run` to run the image

## Makefile
A makefile exists that will help with the following commands:

### Run
Compile and run with `make run`

### Build
Create binaries, upx pack and buld Docker image with `make all`

### Docker Run
Run docker image with `make docker-run`

### Docker Push
Push image to Docker hub with `make docker-push`
