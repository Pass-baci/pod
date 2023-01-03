package main

import (
	"flag"
	"fmt"
	"github.com/Pass-baci/common"
	"github.com/Pass-baci/pod/domain/repository"
	"github.com/Pass-baci/pod/domain/service"
	"github.com/Pass-baci/pod/handler"
	"github.com/Pass-baci/pod/proto/pod"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
	"github.com/go-micro/plugins/v3/registry/consul"
	ratelimiter "github.com/go-micro/plugins/v3/wrapper/ratelimiter/uber"
	microOpentracing "github.com/go-micro/plugins/v3/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net"
	"net/http"
	"path/filepath"
)

var (
	// 注册中心配置
	consulHost       = "192.168.230.135"
	consulPort int64 = 8500
	// 熔断器
	hystrixPort = 9092
	// 监控端口
	prometheusPort = 9192
)

func main() {
	// 配置consul
	consulClient := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{
			fmt.Sprintf("%s:%d", consulHost, consulPort),
		}
	})

	// 配置中心consul
	conf, err := common.GetConsulConfig(consulHost, consulPort, "/micro/config")
	if err != nil {
		common.Fatalf("连接配置中心consul失败 err: %s \n", err.Error())
	}
	common.Info("添加配置中心consul成功")

	// 使用配置中心获取数据库配置
	var mysqlConfig = &common.MysqlConfig{}
	if mysqlConfig, err = common.GetMysqlConfigFromConsul(conf, "mysql"); err != nil {
		common.Fatalf("配置中心获取数据库配置失败 err: %s \n", err.Error())
	}
	common.Info("配置中心获取数据库配置成功")

	// 连接数据库
	var db *gorm.DB
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlConfig.User, mysqlConfig.Pwd, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database)
	if db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}); err != nil {
		common.Fatalf("连接数据库失败 err: %s \n", err.Error())
	}
	common.Info("连接数据库成功")

	// 使用配置中心获取Jaeger配置
	var jaegerConfig = &common.JaegerConfig{}
	if jaegerConfig, err = common.GetJaegerConfigFromConsul(conf, "jaeger-pod"); err != nil {
		common.Fatalf("配置中心获取Jaeger配置失败 err: %s \n", err.Error())
	}
	common.Info("配置中心获取Jaeger配置成功")

	// 添加链路追踪
	tracer, closer, err := common.NewTracer(jaegerConfig.ServiceName, jaegerConfig.Address)
	if err != nil {
		common.Fatalf("添加链路追踪失败 err: %s \n", err.Error())
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
	common.Info("添加链路追踪成功")

	// 添加熔断器
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	common.Info("添加熔断器成功")

	// 启动监听程序
	go func() {
		// http://ip:port/turbine/turbine.stream
		// 看板访问地址：http://192.168.230.135:9002/hystrix
		if err = http.ListenAndServe(net.JoinHostPort("0.0.0.0", "9092"), hystrixStreamHandler); err != nil {
			common.Errorf("启动监听程序失败 err: %s \n", err.Error())
		}
	}()

	// 添加日志中心
	// 需要把程序日志打入到日志文件中
	//common.Info("添加日志中心成功")
	// 在程序代码中添加filebeat.yml文件
	// 启动filebeat 启动命令 ./filebeat -e -c filebeat.yml

	common.PrometheusBoot(prometheusPort)

	// 创建K8s链接
	// 在集群外部使用
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "kubeconfig file 在当前系统中的地址")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig file 在当前系统中的地址")
	}
	flag.Parse()
	// 创建 config 实例
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		common.Fatalf("创建config实例失败 err: %s \n", err.Error())
	}
	common.Info("创建config实例成功")

	// 在集群中使用
	//config, err = rest.InClusterConfig()
	//if err != nil {
	//	common.Fatalf("创建config实例失败 err: %s \n", err.Error())
	//}

	// 创建程序可操作的客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		common.Fatalf("创建k8s客户端失败 err: %s \n", err.Error())
	}
	common.Info("创建k8s客户端成功")

	// Create service
	srv := micro.NewService(
		micro.Server(server.NewServer(func(options *server.Options) {
			options.Advertise = "192.168.230.135:8081"
		})),
		micro.Name("go.micro.service.pod"),
		micro.Version("latest"),
		micro.Address(":8081"),
		// 添加consul
		micro.Registry(consulClient),
		// 添加链路追踪
		micro.WrapHandler(microOpentracing.NewHandlerWrapper(opentracing.GlobalTracer())),
		micro.WrapClient(microOpentracing.NewClientWrapper(opentracing.GlobalTracer())),
		// 只作为客户端的时候起作用
		//micro.WrapClient(hystrix2.NewClientHystrixWrapper()),
		// 添加限流
		micro.WrapHandler(ratelimiter.NewHandlerWrapper(1000)),
	)

	srv.Init()

	podRepository := repository.NewPodRepository(db)
	if err = podRepository.InitTable(); err != nil {
		common.Fatalf("创建数据表失败 err: %s \n", err.Error())
	}
	common.Info("创建数据表成功")

	// Register handler
	pod.RegisterPodHandler(srv.Server(), handler.NewPodHandle(service.NewPodDataService(podRepository, clientset)))

	// Run service
	if err = srv.Run(); err != nil {
		common.Fatalf("Run service失败 err: %s \n", err.Error())
	}
}
