SUBDIRS := $(wildcard */.)
GOALS   := all clean

.PHONY: $(GOALS) $(SUBDIRS)

$(GOALS): $(SUBDIRS)

$(SUBDIRS):
	@$(MAKE) -C $@ $(MAKECMDGOALS)
