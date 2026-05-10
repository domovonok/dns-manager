package client

import (
	"context"
	"fmt"
	"time"

	dnsv1 "github.com/domovonok/dns-manager/api/dns/v1"
	"github.com/domovonok/dns-manager/internal/config"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	addr    string
	timeout time.Duration
}

func New(cfg *config.Client) *Client {
	return &Client{addr: cfg.DefaultAddr, timeout: cfg.RequestTimeout}
}

func (c *Client) Execute() error {
	return c.command().Execute()
}

func (c *Client) command() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "dns-manager",
		Short: "Manage DNS Servers",
	}

	rootCmd.PersistentFlags().StringVar(&c.addr, "addr", c.addr, "gRPC server address")

	rootCmd.AddCommand(c.addCmd(), c.removeCmd(), c.listCmd())

	return rootCmd
}

func (c *Client) addCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <ip>",
		Short: "Add DNS Server",
		Args:  cobra.ExactArgs(1),
		RunE:  c.addDNSServer,
	}
}

func (c *Client) removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <ip>",
		Short: "Remove DNS Server",
		Args:  cobra.ExactArgs(1),
		RunE:  c.removeDNSServer,
	}
}

func (c *Client) listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List DNS Servers",
		Args:  cobra.NoArgs,
		RunE:  c.listDNSServers,
	}
}

func (c *Client) addDNSServer(cmd *cobra.Command, args []string) (err error) {
	ip := args[0]

	return c.withClient(cmd, func(ctx context.Context, client dnsv1.DnsServiceClient) error {
		_, err := client.Add(ctx, &dnsv1.AddRequest{Ip: ip})
		if err != nil {
			return fmt.Errorf("can not add %s: %w", ip, err)
		}
		fmt.Println("added", ip)
		return nil
	})
}

func (c *Client) removeDNSServer(cmd *cobra.Command, args []string) (err error) {
	ip := args[0]

	return c.withClient(cmd, func(ctx context.Context, client dnsv1.DnsServiceClient) error {
		_, err := client.Remove(ctx, &dnsv1.RemoveRequest{Ip: ip})
		if err != nil {
			return fmt.Errorf("can not remove %s: %w", ip, err)
		}
		fmt.Println("removed", ip)
		return nil
	})
}

func (c *Client) listDNSServers(cmd *cobra.Command, _ []string) (err error) {
	return c.withClient(cmd, func(ctx context.Context, client dnsv1.DnsServiceClient) error {
		res, err := client.List(ctx, &emptypb.Empty{})
		if err != nil {
			return fmt.Errorf("can not list dns servers: %w", err)
		}

		for _, ip := range res.Ips {
			fmt.Println(ip)
		}

		return nil
	})
}

func (c *Client) withClient(cmd *cobra.Command, call func(ctx context.Context, client dnsv1.DnsServiceClient) error) (err error) {
	ctx, cancel := context.WithTimeout(cmd.Context(), c.timeout)
	defer cancel()

	conn, err := grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("can not connect to %s: %w", c.addr, err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	return call(ctx, dnsv1.NewDnsServiceClient(conn))
}
