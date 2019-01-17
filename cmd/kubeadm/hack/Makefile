Version =v4.0.1

all: tde

compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src index.tenxcloud.com/tenx_containers/goupx:latest goupx kubeadm

tde: compress
	docker build -t 192.168.1.52/system_containers/tde:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
