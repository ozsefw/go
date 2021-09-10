package logtest

import (
	"os"
	"math/rand"
	"github.com/sirupsen/logrus"
)

func test01(){

	customFormatter := new(logrus.TextFormatter)
	customFormatter.FullTimestamp = true                    // 显示完整时间
	customFormatter.TimestampFormat = "2006-01-02 15:04:05" // 时间格式
	customFormatter.DisableTimestamp = true                // 禁止显示时间
	// customFormatter.DisableColors = false                   // 禁止颜色显示

	customFormatter.ForceColors = true
	
	logrus.SetFormatter(customFormatter)
	logrus.SetOutput(os.Stdout)

	logrus.SetLevel(logrus.TraceLevel)

	logrus.Trace("trace msg")
	logrus.Debug("debug msg")
	logrus.Info("info msg")
	logrus.Warn("warn msg")
	logrus.Error("Error msg")

	n := 10
	logrus.Tracef("value N: %v\n", n)
}

func test02(){
	customFormatter := new(logrus.TextFormatter)
	customFormatter.DisableTimestamp = true                // 禁止显示时间
	customFormatter.ForceColors = true

	logrus.SetFormatter(customFormatter)
	// logrus.SetOutput(os.Stdout)
	// logrus.SetLevel(logrus.TraceLevel)

	min, max := -1, -1
	for i:=0; i<=1000; i++{
		n := (rand.Int()%100)+1
		if min == -1 || max == -1{
			min = n
			max = n
		}
		if n < min{
			min = n
		}
		if n > max{
			max = n
		}
		// logrus.Infof("n: %v", n)
	}

	logrus.Infof("[%v, %v]", min, max)
}

func Run(){
	test02()
}