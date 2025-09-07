/*
Created on Fri Sep 16 17:04:36 2024
@author:liwancai

	QQ:248411282
	Tel:13199701121
*/
package RunInfo

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gitlab.liwancai.com/liwancai/EQUseApi"
	"gitlab.liwancai.com/liwancai/GoScripts/Formulae"
	"gitlab.liwancai.com/liwancai/Logger"
)

var log *Logger.LogN

func SetLogger(l *Logger.LogN) {
	log = l //配置log信息
}

type RunTimeBaseN struct {
	RunTime *Formulae.RunTimeN
	IP      string
	Mac     string
}

func RunTimeInit() *RunTimeBaseN {
	return &RunTimeBaseN{}
}
func (app *RunTimeBaseN) Start() {
	app.RunTime = Formulae.RunTime() //用于计算业务逻辑运行时间
	app.IP = GetLocalLANIP()
	app.Mac = GetLocalLANMac()
	msginfo := fmt.Sprintf(
		"\n■|服务器【%s】启动\n■|Mac: %s\n■|程序: %s\n■|时间: %v",
		app.IP, app.Mac, filepath.Base(os.Args[0]),
		app.RunTime.Start.Format("20060102 15:04:05"))
	EQUseApi.SendTxT(msginfo, EQUseApi.SendToGroupList, []string{}, []string{})
	log.Info(msginfo)
	go ExceptErr("程序中途退出")
}
func (app *RunTimeBaseN) Stop() {
	value, unit := app.RunTime.EndRun()
	msginfo := fmt.Sprintf(
		"\n■|服务器【%s】完成\n■|Mac: %s\n■|程序: %s\n■|耗时: %.3f %s",
		app.IP, app.Mac, filepath.Base(os.Args[0]),
		value, unit)
	log.Info(msginfo)
	EQUseApi.SendTxT(msginfo, EQUseApi.SendToGroupList, []string{}, []string{})
}

// 获取第一个活动的非回环网络接口的MAC地址
func GetLocalLANMac() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("获取网络接口出错: %v", err)
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				return mac
			}
		}
	}
	return ""
}

func GetLocalLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		errmsg := fmt.Sprintf("获取网络接口出错:%s", err)
		log.Error(errmsg)
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 { // 过滤掉非物理网卡和无效网卡
			addrs, err := iface.Addrs()
			if err != nil {
				errmsg := fmt.Sprintf("获取IP地址出错:%s", err)
				log.Error(errmsg)
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					return ipNet.IP.String() // 获取IPv4地址
				}
			}
		}
	}
	return ""
}

// 捕获错误异常处理
func ExceptErr(msg string) {
	if err := recover(); err != nil {
		lcip := GetLocalLANIP()
		err = ErrCase(err)
		EQUseApi.SendLogTxT(fmt.Sprintf("■|服务器【%s】警告\n■|程序: %s\n■|信息: %s \n■|错误: %s",
			lcip, filepath.Base(os.Args[0]), msg, err), EQUseApi.SendToGroupList)
		log.Debug(fmt.Sprintf("异常错误信号捕捉: %v", err))
		log.FlushCache()
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)
	sig := <-c
	log.Info(fmt.Sprintf("接收到终止信号: %v", sig))
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func ErrCase(err interface{}) string {
	errmsg := fmt.Sprintf("%s", err)
	if strings.Contains(errmsg, "invalid memory address or nil pointer dereference") {
		errmsg = `在Go语言中遇到了空指针解引用或无效内存地址,请检查代码中涉及指针的部分.
	如: 配置文件内容参数是否不全;是否存在import的库文件没有SetLogger设置;`
	}
	log.Error(errmsg)
	return errmsg
}
