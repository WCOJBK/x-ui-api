package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"x-ui/logger"
	"x-ui/util/common"

	"github.com/xtls/xray-core/app/proxyman/command"
	statsService "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/shadowsocks_2022"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// XrayAPI 结构体定义了与 Xray API 交互的客户端
type XrayAPI struct {
	HandlerServiceClient *command.HandlerServiceClient // 处理入站和出站的服务客户端
	StatsServiceClient   *statsService.StatsServiceClient // 统计信息服务客户端
	grpcClient           *grpc.ClientConn // gRPC 连接
	isConnected          bool // 连接状态
}

// Init 初始化 XrayAPI，连接到指定的 API 端口
func (x *XrayAPI) Init(apiPort int) error {
	if apiPort <= 0 {
		return fmt.Errorf("invalid Xray API port: %d", apiPort) // 检查端口有效性
	}

	addr := fmt.Sprintf("127.0.0.1:%d", apiPort) // 构建连接地址
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials())) // 建立 gRPC 连接
	if err != nil {
		return fmt.Errorf("failed to connect to Xray API: %w", err) // 连接失败处理
	}

	x.grpcClient = conn // 保存连接
	x.isConnected = true // 更新连接状态

	// 创建服务客户端
	hsClient := command.NewHandlerServiceClient(conn)
	ssClient := statsService.NewStatsServiceClient(conn)

	x.HandlerServiceClient = &hsClient // 保存处理服务客户端
	x.StatsServiceClient = &ssClient // 保存统计服务客户端

	return nil // 初始化成功
}

// Close 关闭 XrayAPI 的 gRPC 连接
func (x *XrayAPI) Close() {
	if x.grpcClient != nil {
		x.grpcClient.Close() // 关闭连接
	}
	x.HandlerServiceClient = nil // 清空客户端
	x.StatsServiceClient = nil // 清空统计客户端
	x.isConnected = false // 更新连接状态
}

// AddInbound 添加入站配置
func (x *XrayAPI) AddInbound(inbound []byte) error {
	client := *x.HandlerServiceClient // 获取处理服务客户端

	conf := new(conf.InboundDetourConfig) // 创建入站配置对象
	err := json.Unmarshal(inbound, conf) // 解析入站配置 JSON
	if err != nil {
		logger.Debug("Failed to unmarshal inbound:", err) // 解析失败日志
		return err
	}
	config, err := conf.Build() // 构建入站配置
	if err != nil {
		logger.Debug("Failed to build inbound Detour:", err) // 构建失败日志
		return err
	}
	inboundConfig := command.AddInboundRequest{Inbound: config} // 创建添加入站请求

	_, err = client.AddInbound(context.Background(), &inboundConfig) // 发送添加入站请求
	return err // 返回结果
}

// DelInbound 删除指定标签的入站配置
func (x *XrayAPI) DelInbound(tag string) error {
	client := *x.HandlerServiceClient // 获取处理服务客户端
	_, err := client.RemoveInbound(context.Background(), &command.RemoveInboundRequest{
		Tag: tag, // 指定要删除的入站标签
	})
	return err // 返回结果
}

// AddUser 向指定入站添加用户
func (x *XrayAPI) AddUser(Protocol string, inboundTag string, user map[string]interface{}) error {
	var account *serial.TypedMessage // 用户账户信息

	// 根据协议类型创建不同的账户信息
	switch Protocol {
	case "vmess":
		account = serial.ToTypedMessage(&vmess.Account{
			Id: user["id"].(string), // 获取用户 ID
		})
	case "vless":
		account = serial.ToTypedMessage(&vless.Account{
			Id:   user["id"].(string),
			Flow: user["flow"].(string), // 获取流信息
		})
	case "trojan":
		account = serial.ToTypedMessage(&trojan.Account{
			Password: user["password"].(string), // 获取密码
		})
	case "shadowsocks":
		var ssCipherType shadowsocks.CipherType // 定义加密类型
		switch user["cipher"].(string) {
		case "aes-128-gcm":
			ssCipherType = shadowsocks.CipherType_AES_128_GCM
		case "aes-256-gcm":
			ssCipherType = shadowsocks.CipherType_AES_256_GCM
		case "chacha20-poly1305", "chacha20-ietf-poly1305":
			ssCipherType = shadowsocks.CipherType_CHACHA20_POLY1305
		case "xchacha20-poly1305", "chacha20-ietf-poly1305":
			ssCipherType = shadowsocks.CipherType_XCHACHA20_POLY1305
		default:
			ssCipherType = shadowsocks.CipherType_NONE
		}

		// 根据加密类型创建账户信息
		if ssCipherType != shadowsocks.CipherType_NONE {
			account = serial.ToTypedMessage(&shadowsocks.Account{
				Password:   user["password"].(string),
				CipherType: ssCipherType,
			})
		} else {
			account = serial.ToTypedMessage(&shadowsocks_2022.ServerConfig{
				Key:   user["password"].(string),
				Email: user["email"].(string),
			})
		}
	default:
		return nil // 不支持的协议类型
	}

	client := *x.HandlerServiceClient // 获取处理服务客户端

	// 发送添加用户请求
	_, err := client.AlterInbound(context.Background(), &command.AlterInboundRequest{
		Tag: inboundTag, // 指定入站标签
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: &protocol.User{
				Email:   user["email"].(string), // 获取用户邮箱
				Account: account, // 设置账户信息
			},
		}),
	})
	return err // 返回结果
}

// RemoveUser 从指定入站中移除用户
func (x *XrayAPI) RemoveUser(inboundTag, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 设置上下文超时
	defer cancel() // 确保在函数结束时取消上下文

	op := &command.RemoveUserOperation{Email: email} // 创建移除用户操作
	req := &command.AlterInboundRequest{
		Tag:       inboundTag, // 指定入站标签
		Operation: serial.ToTypedMessage(op), // 设置操作
	}

	_, err := (*x.HandlerServiceClient).AlterInbound(ctx, req) // 发送移除用户请求
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err) // 处理错误
	}

	return nil // 成功
}

// GetTraffic 获取流量统计信息
func (x *XrayAPI) GetTraffic(reset bool) ([]*Traffic, []*ClientTraffic, error) {
	if x.grpcClient == nil {
		return nil, nil, common.NewError("xray api is not initialized") // 检查 API 是否初始化
	}

	// 正则表达式用于匹配流量统计信息
	trafficRegex := regexp.MustCompile(`(inbound|outbound)>>>([^>]+)>>>traffic>>>(downlink|uplink)`)
	clientTrafficRegex := regexp.MustCompile(`user>>>([^>]+)>>>traffic>>>(downlink|uplink)`)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10) // 设置上下文超时
	defer cancel() // 确保在函数结束时取消上下文

	if x.StatsServiceClient == nil {
		return nil, nil, common.NewError("xray StatusServiceClient is not initialized") // 检查统计服务客户端是否初始化
	}

	resp, err := (*x.StatsServiceClient).QueryStats(ctx, &statsService.QueryStatsRequest{Reset_: reset}) // 查询统计信息
	if err != nil {
		logger.Debug("Failed to query Xray stats:", err) // 查询失败日志
		return nil, nil, err
	}

	tagTrafficMap := make(map[string]*Traffic) // 存储入站和出站流量的映射
	emailTrafficMap := make(map[string]*ClientTraffic) // 存储用户流量的映射

	// 处理统计信息
	for _, stat := range resp.GetStat() {
		if matches := trafficRegex.FindStringSubmatch(stat.Name); len(matches) == 4 {
			processTraffic(matches, stat.Value, tagTrafficMap) // 处理流量信息
		} else if matches := clientTrafficRegex.FindStringSubmatch(stat.Name); len(matches) == 3 {
			processClientTraffic(matches, stat.Value, emailTrafficMap) // 处理用户流量信息
		}
	}
	return mapToSlice(tagTrafficMap), mapToSlice(emailTrafficMap), nil // 返回流量统计结果
}

// processTraffic 处理入站和出站流量信息
func processTraffic(matches []string, value int64, trafficMap map[string]*Traffic) {
	isInbound := matches[1] == "inbound" // 判断是入站还是出站
	tag := matches[2] // 获取标签
	isDown := matches[3] == "downlink" // 判断是下行还是上行流量

	if tag == "api" {
		return // 忽略 API 流量
	}

	traffic, ok := trafficMap[tag] // 获取流量信息
	if !ok {
		traffic = &Traffic{
			IsInbound:  isInbound,
			IsOutbound: !isInbound,
			Tag:        tag,
		}
		trafficMap[tag] = traffic // 添加到流量映射
	}

	if isDown {
		traffic.Down = value // 设置下行流量
	} else {
		traffic.Up = value // 设置上行流量
	}
}

// processClientTraffic 处理用户流量信息
func processClientTraffic(matches []string, value int64, clientTrafficMap map[string]*ClientTraffic) {
	email := matches[1] // 获取用户邮箱
	isDown := matches[2] == "downlink" // 判断是下行还是上行流量

	traffic, ok := clientTrafficMap[email] // 获取用户流量信息
	if !ok {
		traffic = &ClientTraffic{Email: email} // 创建新的用户流量信息
		clientTrafficMap[email] = traffic // 添加到用户流量映射
	}

	if isDown {
		traffic.Down = value // 设置下行流量
	} else {
		traffic.Up = value // 设置上行流量
	}
}

// mapToSlice 将映射转换为切片
func mapToSlice[T any](m map[string]*T) []*T {
	result := make([]*T, 0, len(m)) // 创建切片
	for _, v := range m {
		result = append(result, v) // 添加映射中的值到切片
	}
	return result // 返回切片
}

// RouteInboundToOutbound 将指定的入站流量路由到指定的出站
func (x *XrayAPI) RouteInboundToOutbound(inboundTag string, outboundTag string) error {
	client := *x.HandlerServiceClient // 获取处理服务客户端

	// 创建路由请求
	routeRequest := &command.RouteRequest{
		InboundTag:  inboundTag,  // 指定入站标签
		OutboundTag: outboundTag,  // 指定出站标签
	}

	// 发送路由请求
	_, err := client.Route(context.Background(), routeRequest)
	if err != nil {
		logger.Debug("Failed to route inbound to outbound:", err) // 路由失败日志
		return err // 返回错误
	}

	return nil // 成功
}
