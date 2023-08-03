package simplegroupcache

import (
	"context"
	"fmt"
	"simple-groupcache/pb"
	"simple-groupcache/registry"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// client 模块实现节点访问其他远程节点 从而获取缓存的能力

type client struct {
	name string // 服务名称 pcache/ip:addr
}

func NewClient(service string) *client {
	return &client{name: service}
}

// Fetch 从remote peer获取对应缓存值
func (c *client) Fetch(group string, key string) ([]byte, error) {
	// 创建一个etcd client
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	// 发现服务 取得与服务的连接
	conn, err := registry.EtcdDial(cli, c.name)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 创建grpc client
	grpcClient := pb.NewGroupcacheClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 发送请求
	resp, err := grpcClient.Get(ctx, &pb.GetRequest{
		Group: group,
		Key:   key,
	})
	if err != nil {
		return nil, fmt.Errorf("could not get %s/%s from peer %s", group, key, c.name)
	}

	return resp.GetValue(), nil
}

// 测试Client是否实现了Fetcher接口
var _ Fetcher = (*client)(nil)
