# MCP Hub 构建系统
# 自动发现 services/*/cmd/* 下的所有独立二进制
# 新增服务只需创建目录结构，无需修改此文件

GO        := go
BINDIR    := bin
LDFLAGS   := -ldflags="-s -w"
CGO_ENABLED := 0

# ─── 自动发现所有二进制 ────────────────────────────────────────────

SERVER_BIN := $(BINDIR)/mcp-server

# 自动发现 services/*/cmd/*/main.go
SERVICE_CMDS := $(wildcard services/*/cmd/*/main.go)
# 从路径中提取二进制名：services/workday/cmd/workday-svc/main.go → workday-svc
SERVICE_BINS := $(foreach f,$(SERVICE_CMDS),$(BINDIR)/$(notdir $(patsubst %/,%,$(dir $(f)))))

# 全部二进制
ALL_BINS := $(SERVER_BIN) $(SERVICE_BINS)

# ─── 生成构建规则 ──────────────────────────────────────────────────
# 为每个服务生成独立的构建规则，避免 Make 通配符匹配歧义
# services/workday/cmd/workday-svc/main.go → bin/workday-svc
define GEN_SERVICE_RULE
$(BINDIR)/$(notdir $(patsubst %/,%,$(dir $(1)))): $(1)
	@mkdir -p $(BINDIR)
	$$(info 构建服务: $(notdir $(patsubst %/,%,$(dir $(1))))...)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $$@ ./$(dir $(1))
endef

# 生成所有服务规则
$(foreach cmd,$(SERVICE_CMDS),$(eval $(call GEN_SERVICE_RULE,$(cmd))))

# ─── 目标 ─────────────────────────────────────────────────────────

.PHONY: all build clean test lint list

all: build

# 构建所有二进制
build: $(ALL_BINS)

# 构建主服务
$(SERVER_BIN): cmd/server/main.go
	@mkdir -p $(BINDIR)
	$(info 构建主服务...)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $@ ./cmd/server

# 构建单个服务（如 make build/server、make build/workday-svc）
build/%: $(BINDIR)/%
	@:

# 清理构建产物
clean:
	rm -rf $(BINDIR)

# 运行所有测试
test:
	$(GO) test ./...

# 代码检查
lint:
	$(GO) vet ./...

# 列出所有可构建的二进制
list:
	@echo "可构建的二进制:"
	@echo "  $(SERVER_BIN) (主服务)"
	@for bin in $(SERVICE_BINS); do echo "  $$bin"; done

# ─── 信息 ─────────────────────────────────────────────────────────

$(info ╔══════════════════════════════════════════╗)
$(info ║  MCP Hub 构建系统                        ║)
$(info ╠══════════════════════════════════════════╣)
$(info ║  主服务: $(SERVER_BIN))
$(info ║  服务二进制: $(words $(SERVICE_BINS)) 个)
$(info ║  全部二进制: $(words $(ALL_BINS)) 个)
$(info ╠══════════════════════════════════════════╣)
$(info ║  用法:                                   ║)
$(info ║    make build             全部构建        ║)
$(info ║    make build/server      仅主服务        ║)
$(info ║    make build/workday-svc 单个服务        ║)
$(info ║    make clean             清理            ║)
$(info ║    make list              列出二进制      ║)
$(info ╚══════════════════════════════════════════╝)
$(info )