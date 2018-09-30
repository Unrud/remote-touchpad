ARCH = $(shell flatpak --default-arch)

APP_ID = com.github.unrud.RemoteTouchpad
APP_NAME = RemoteTouchpad

.PHONY: all
all: bundle

.PHONY: build
build:
	flatpak-builder --arch=$(ARCH) --force-clean $(APP_NAME) $(APP_ID).json

.PHONY: repo
repo:
	flatpak-builder --arch=$(ARCH) --force-clean $(APP_NAME) $(APP_ID).json --repo=repo

.PHONY: bundle
bundle: repo
	flatpak build-bundle --arch=$(ARCH) repo $(APP_NAME)-$(ARCH).flatpak $(APP_ID)

.PHONY: clean
clean:
	rm -rf $(APP_NAME) $(APP_NAME)-*.flatpak repo .flatpak-builder || true

.PHONY: install
install:
	flatpak install --user $(APP_NAME)-$(ARCH).flatpak

.PHONY: uninstall
uninstall:
	flatpak uninstall --user $(APP_ID)
