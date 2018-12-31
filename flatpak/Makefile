ARCH = $(shell flatpak --default-arch)

APP_ID = com.github.unrud.RemoteTouchpad
APP_NAME = $(shell echo $(APP_ID) | rev | cut --delimiter=. --fields=1 | rev)
FORMAT = $(shell ls -1 $(APP_ID).json $(APP_ID).yaml 2>/dev/null | tail -n1 | rev | cut -d. -f1 | rev)
BRANCH = master

.PHONY: all
all: bundle

.PHONY: build
build:
	mkdir -p build
	flatpak-builder --arch=$(ARCH) --force-clean build/$(APP_NAME) $(APP_ID).$(FORMAT)

.PHONY: repo
repo:
	mkdir -p build repo
	flatpak-builder --arch=$(ARCH) --force-clean build/$(APP_NAME) $(APP_ID).$(FORMAT) --repo=repo/$(APP_NAME)

.PHONY: bundle
bundle: repo
	flatpak build-bundle --arch=$(ARCH) repo/$(APP_NAME) $(APP_NAME)-$(ARCH).flatpak $(APP_ID) $(BRANCH)

.PHONY: clean
clean:
	rm -rf $(APP_NAME)-*.flatpak build/$(APP_NAME) repo/$(APP_NAME) .flatpak-builder || true
	rmdir build || true
	rmdir repo || true

.PHONY: install
install:
	flatpak install --user $(APP_NAME)-$(ARCH).flatpak

.PHONY: uninstall
uninstall:
	flatpak uninstall --user $(APP_ID)/$(ARCH)/$(BRANCH)
