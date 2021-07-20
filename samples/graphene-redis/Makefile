# Build Redis as follows:
#
# - make               -- create non-SGX no-debug-log manifest
# - make SGX=1         -- create SGX no-debug-log manifest
# - make SGX=1 DEBUG=1 -- create SGX debug-log manifest
#
# Any of these invocations clones Redis' git repository and builds Redis in
# default configuration and in the latest-to-date (5.0.5) version.
#
# By default, Redis uses poll/epoll mechanism of Linux. To build Redis with
# select, use `make USE_SELECT=1`. For correct re-builds, always clean up
# Redis source code beforehand via `make distclean`.
#
# Use `make clean` to remove Graphene-generated files and `make distclean` to
# additionally remove the cloned Redis git repository.

################################# CONSTANTS ###################################


SRCDIR = src
COMMIT = 6.0.5
TAR_SHA256 = f7ded6c27d48c20bc78e797046c79b6bc411121f0c2d7eead9fea50d6b0b6290

ifeq ($(DEBUG),1)
GRAPHENE_LOG_LEVEL = debug
else
GRAPHENE_LOG_LEVEL = error
endif

.PHONY: all
all: redis-server redis-server.manifest
ifeq ($(SGX),1)
all: redis-server.manifest.sgx redis-server.sig redis-server.token
endif

# Download files required for Makefile include
# These are the latest include files as of v1.12-rc1
Makefile.Host:
	wget "https://github.com/oscarlab/graphene/raw/06ec21be3288730f318fd99f1c70e48af25612c1/Scripts/Makefile.Host"

Makefile.configs: Makefile.Host
	wget "https://github.com/oscarlab/graphene/raw/5cb0389f56c2d1a91511fe411cca0aa914d1ff04/Scripts/Makefile.configs"

include Makefile.configs

############################## REDIS EXECUTABLE ###############################

# Redis is built as usual, without any changes to the build process (except to
# test select syscall instead of poll/epoll). The source is downloaded from the
# GitHub repo (5.0.5 tag) and built via `make`. The result of this build process
# is the final executable "src/redis-server".

$(SRCDIR)/Makefile:
	wget -O redis.tar.gz https://github.com/antirez/redis/archive/$(COMMIT).tar.gz
	echo "$(TAR_SHA256) redis.tar.gz" | sha256sum -c
	mkdir $(SRCDIR)
	tar -C $(SRCDIR) --strip-components=1 -xf redis.tar.gz

ifeq ($(USE_SELECT),1)
$(SRCDIR)/src/redis-server: $(SRCDIR)/Makefile
	sed -i 's|#define HAVE_EPOLL 1|/* no HAVE_EPOLL */|g' src/src/config.h
	$(MAKE) -C $(SRCDIR)
else
$(SRCDIR)/src/redis-server: $(SRCDIR)/Makefile
	$(MAKE) -C $(SRCDIR)
endif

################################ REDIS MANIFEST ###############################

# The template file contains almost all necessary information to run Redis
# under Graphene / Graphene-SGX. We create redis.manifest (to be run under
# non-SGX Graphene) by simply replacing variables in the template file via sed.

redis-server.manifest: redis-server.manifest.template
	graphene-manifest \
		-Dlog_level=$(GRAPHENE_LOG_LEVEL) \
		-Darch_libdir=$(ARCH_LIBDIR) \
		$< > $@

# Manifest for Graphene-SGX requires special "graphene-sgx-sign" procedure. This procedure measures
# all Redis trusted files, adds the measurement to the resulting manifest.sgx file (among other,
# less important SGX options) and creates redis-server.sig (SIGSTRUCT object).
#
# Graphene-SGX requires EINITTOKEN and SIGSTRUCT objects (see SGX hardware ABI, in particular EINIT
# instruction). The "graphene-sgx-get-token" script generates EINITTOKEN based on a SIGSTRUCT and
# puts it in .token file. Note that filenames must be the same as the manifest name (i.e.,
# "redis-server"). EINITTOKEN must be generated on the machine where the application will run, not
# where it was built.

# Make on Ubuntu <= 20.04 doesn't support "Rules with Grouped Targets" (`&:`),
# we need to hack around.
redis-server.sig redis-server.manifest.sgx: sgx_outputs
	@:

.INTERMEDIATE: sgx_outputs
sgx_outputs: redis-server.manifest $(SRCDIR)/src/redis-server
	openssl genrsa -3 -out signer-key.pem 3072
	graphene-sgx-sign \
		--key signer-key.pem \
		--manifest redis-server.manifest \
		--output redis-server.manifest.sgx

redis-server.token: redis-server.sig
	graphene-sgx-get-token --output $@ --sig $<

########################### COPIES OF EXECUTABLES #############################

# Redis build process creates the final executable as src/redis-server. For
# simplicity, copy it into our root directory.

redis-server: $(SRCDIR)/src/redis-server
	cp $< $@

############################## RUNNING TESTS ##################################
.PHONY: start-native-server
start-native-server: all
	./redis-server --save '' --protected-mode no

ifeq ($(SGX),)
GRAPHENE = graphene-direct
else
GRAPHENE = graphene-sgx
endif

.PHONY: start-graphene-server
start-graphene-server: all
	$(GRAPHENE) redis-server --save '' --protected-mode no

################################## CLEANUP ####################################

.PHONY: clean
clean:
	$(RM) *.token *.sig *.manifest.sgx *.manifest redis-server *.rdb

.PHONY: distclean
distclean: clean
	$(RM) -r $(SRCDIR) redis.tar.gz
