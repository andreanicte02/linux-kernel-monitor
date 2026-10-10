KDIR ?= /lib/modules/$(shell uname -r)/build

.PHONY: all clean load unload

all:
	$(MAKE) -C $(KDIR) M=$(CURDIR) modules

clean:
	$(MAKE) -C $(KDIR) M=$(CURDIR) clean

load: all
	sudo insmod kmon.ko

unload:
	sudo rmmod kmon
