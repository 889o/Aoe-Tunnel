package service

import _ "embed"

// installAgentScript 是通用一键安装脚本本体：既可以直接托管到云存储，
// 也可以由面板在头部注入 AOE_* 变量后整体下发，两者行为完全一致。
//
//go:embed install_agent.sh
var installAgentScript string
