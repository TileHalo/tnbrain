all: tnbrain

tnbrain:
	env GOOS=linux GOARCH=arm GOARM=5 go build

.PHONY: all tnbrain
