package aco

import (
	"errors"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"sync"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

const CHUNCK_SIZE int = 100*256

const STATE_UNFILLED int = 0
const STATE_DOWNLOADING int = 1
const STATE_FILLED int = 2

type TaskInfo struct {
	chunck_id int
	node_id int
	state string
	download_rate int
}

type TaskRecord struct{
	chunck_id int
	node_id int
	start_time time.Time
	finish_time time.Time
	download_rate int
	state string	// "running", "finish"
}

type NodeAttribute struct{
	chunck_bitvec []bool
	bandwidth int
	used_bandwidth int
}

type DownloadTask struct{
	chunck_count int
	node_count int

	chunck_state_list []int

	chunck_route_table map[int]NodeAttribute

	msg_ch chan TaskInfo
	download_flow map[string]TaskRecord // key : "${chunck_id},${node_id}"

	bandwidth int
	// free_banawidth int

	rt_manager RouteTableManager

	json_file string
	max_conn_count int

	wg sync.WaitGroup
}

func NewDownloadTask(node_count int, chunck_count int) DownloadTask{

	node_id_list := make([]int, node_count)
	for i:=0; i<node_count;i++{
		node_id_list[i] = i
	}

	chunck_route_table := GenerateRandomChunckTable(node_id_list, chunck_count)
	chunck_state_list := make([]int, chunck_count)
	for i:=0; i<chunck_count; i++{
		chunck_state_list[i] = STATE_UNFILLED
	}

	msg_ch := make(chan TaskInfo, 100)
	download_flow := make(map[string]TaskRecord)

	bandwidth := 10_000
	average_rate := 3_000
	rt_manager := NewRouteTableManager(node_id_list, average_rate, bandwidth)
	json_file := "./data/download_flow.json"
	max_conn_count := 3

	wg := sync.WaitGroup{}

	// speed_record := make([][2]int64,0)

	// free_bandwidth := bandwidth
	return DownloadTask{
		chunck_count,
		node_count,

		chunck_state_list,

		chunck_route_table,

		msg_ch,
		download_flow,

		bandwidth,
		// free_bandwidth,
		rt_manager,
		json_file,
		max_conn_count,
		wg,
		// speed_record,
	}
}

func (dt *DownloadTask) Run(){
	go dt.CollectMsg()

	dt.wg.Add(1)
	go dt.collectSpeedDataAndDraw()
	for{
		cur_time := time.Now()
		filled_chunck_count := dt.getFilledChunckCount()
		// fmt.Printf("filled: %v\n", filled_chunck_count)
		if filled_chunck_count >= dt.chunck_count{
			fmt.Printf("Download: %v x %v KBytes\n", filled_chunck_count, CHUNCK_SIZE)
			// dt.saveDownloadFlow()
			// println("Download Finish")
			break
		}

		cur_conn_count := dt.getConnectionCount()
		local_free_bandwidth := dt.getLocalFreeBandwidth()

		fmt.Printf("%02d:%02d                              speed: %v KB/s conn: %v\n", 
			cur_time.Minute(), cur_time.Second(), dt.bandwidth-local_free_bandwidth, cur_conn_count)
		if local_free_bandwidth < 20{
			time.Sleep(time.Duration(500)*time.Millisecond)
			continue
		}

		if cur_conn_count >= dt.max_conn_count{
			time.Sleep(time.Duration(500)*time.Millisecond)
			continue
		}

		next_chunck_id, err := dt.getNextUnfilledChunckId()
		if err != nil{
			time.Sleep(time.Duration(500)*time.Millisecond)
			continue
		}

		// dt.startDownloadChunck(next_chunck_id, local_free_bandwidth)
		available_node_list := dt.getAvailableNodeList(next_chunck_id)

		// 对比测试，这里可以随机选择节点
		select_node_id := dt.rt_manager.SelectACONode(available_node_list)

		remote_free_bandwidth, download_delay := dt.getRemoteBandwidthAndDelay(select_node_id)

		var download_rate int
		if local_free_bandwidth > remote_free_bandwidth{
			download_rate = remote_free_bandwidth
		}else{
			download_rate = local_free_bandwidth
		}
		go StartDownloadTask(next_chunck_id, select_node_id, download_rate, download_delay, dt.msg_ch)

		time.Sleep(time.Duration(500)*time.Millisecond)
	}
	dt.wg.Wait()
}

func (dt *DownloadTask) collectSpeedDataAndDraw(){
	speed_record := make([][2]int64, 0)

	// min_speed, max_speed := 0, 0
	// start_ts, end_ts := 0, 0
	var start_ts, end_ts int64

	i := 0
	for{
		ts := time.Now().UnixNano()/1e6
		if start_ts == 0{
			start_ts = ts
		}
		cur_speed := dt.bandwidth - dt.getLocalFreeBandwidth()
		// if min_speed == 0 || max_speed == 0{
		// 	min_speed = cur_speed
		// 	max_speed = cur_speed
		// }else{
		// 	if cur_speed > max_speed{
		// 		max_speed = cur_speed
		// 	}
		// 	if cur_speed < min_speed{
		// 		min_speed = cur_speed
		// 	}
		// }

		new_speed := [2]int64{ts, int64(cur_speed)}
		speed_record = append(speed_record, new_speed)

		if i>5 {
			if dt.getFilledChunckCount() >= dt.chunck_count{
				end_ts = ts
				break
			}
			i = 0
		}

		time.Sleep(time.Duration(500)*time.Millisecond)
		i += 1
	}

	// dt.drawDownloadSpeedMap(speed_record, min_speed, max_speed) 

	points := plotter.XYs{}
	for _, v := range speed_record{
		points = append(points, plotter.XY{
			X: float64(v[0]),
			Y: float64(v[1]),
		})
	}

	plt := plot.New()
	plt.Y.Min, plt.Y.Max = float64(0), float64(dt.bandwidth)
	// plt.Y.Min, plt.Y.Max = float64(0), float64(10000)
	plt.X.Min, plt.X.Max = float64(start_ts-1000), float64(end_ts+1000)

	if err := plotutil.AddLinePoints(plt,
		"speed", points,
	); err != nil{
		panic(err)
	}

	if err := plt.Save(10*vg.Inch, 5*vg.Inch, "speed_record.png"); err != nil{
		panic(err)
	}

	println("figure done!")

	dt.wg.Done()
}

// func (dt *DownloadTask) drawDownloadSpeedMap(speed_record [][2]int64, min_speed int, max_speed int){
// }

func (dt *DownloadTask) getConnectionCount() int{
	conn_count := 0

	for _, v := range dt.download_flow{
		task_state := v.state
		if task_state == "running"{
			conn_count += 1
		}
	}
	return conn_count
}

type DownloadRecord struct{
	Bandwidth int64
	Start_ts int64
	End_ts int64
	Task_vec [][4]int64
}

func (dt *DownloadTask) saveDownloadFlow(){
	var start_ts, end_ts int64
	task_vec := make([][4]int64, dt.chunck_count)

	for _,v := range dt.download_flow{
		cur_start_ts := v.start_time.UnixNano()/1e6
		cur_end_ts := v.finish_time.UnixNano()/1e6

		if start_ts == 0{
			start_ts = cur_start_ts
		}
		if end_ts == 0{
			end_ts = cur_end_ts
		}

		if cur_start_ts < start_ts{
			start_ts = cur_start_ts
		}
		if cur_end_ts > end_ts{
			end_ts = cur_end_ts
		}

		cur_speed := int64(v.download_rate)
		cur_node_id := int64(v.node_id)

		new_task := [4]int64{cur_start_ts, cur_end_ts, cur_speed, cur_node_id}
		// task_vec = append(task_vec, new_task)
		chunck_id := v.chunck_id
		task_vec[chunck_id] = new_task
	}

	download_record := DownloadRecord{int64(dt.bandwidth), start_ts, end_ts, task_vec}
	json_data, err := json.MarshalIndent(download_record, "", "    ")
	// json_data, err := json.Marshal(download_record)
	if err != nil{
		fmt.Printf("json error: %v\n", err.Error())
	}
	_ = ioutil.WriteFile(dt.json_file, json_data, 0644)
}

// func (dt *DownloadTask) startDownloadChunck(chunck_id int){
// }

func (dt *DownloadTask) getFilledChunckCount() int{
	filled_count := 0
	for _, chunck_state := range dt.chunck_state_list{
		if chunck_state == STATE_FILLED{
			filled_count += 1
		}
	}
	return filled_count
}

func (dt *DownloadTask) getNextUnfilledChunckId() (int, error){
	for chunck_id, chunck_state := range dt.chunck_state_list {
		if chunck_state == STATE_UNFILLED{
			return chunck_id, nil
		}
	}
	return -1, errors.New("no unfilled chunck")
}

// 怎么初始化数据分布表格
// 怎么开始下载流程
// 通过什么方式记录下载流水
// 怎么通过下载流水绘制下载速度图
// 怎么获取local剩余的可用的网速

func (dt *DownloadTask) getAvailableNodeList(chunck_index int) []int{
	node_list := make([]int, 0)

	for node_id, node_attr := range dt.chunck_route_table{
		if node_attr.chunck_bitvec[chunck_index] {
			node_list = append(node_list, node_id)
		}
	}
	return node_list
}

func (dt *DownloadTask) getLocalFreeBandwidth() int{
	free_bandwidth := dt.bandwidth
	for _, task_record := range dt.download_flow{
		if task_record.state == "running"{
			free_bandwidth -= task_record.download_rate
			if free_bandwidth <= 0{
				break
			}
		}
	}

	return free_bandwidth
}

func (dt *DownloadTask) getRemoteBandwidthAndDelay(node_id int) (int, int){
	remote_bandwidth := dt.chunck_route_table[node_id].bandwidth
	return remote_bandwidth, 0
}

func (dt *DownloadTask) CollectMsg(){
	for{
		task_info := <- dt.msg_ch

		chunck_id, node_id, download_rate := task_info.chunck_id, task_info.node_id, task_info.download_rate
		task_index := fmt.Sprintf("%v, %v", task_info.chunck_id, task_info.node_id)

		if task_info.state == "start"{
			start_time := time.Now()
			finish_time := time.Unix(0,0)
			dt.download_flow[task_index] = TaskRecord{
				chunck_id,
				node_id,
				start_time, 
				finish_time, 
				download_rate, 
				"running"}

			dt.updateRouteTable("start", node_id, download_rate)
			dt.chunck_state_list[chunck_id] = STATE_DOWNLOADING
		}else{
			exist_record := dt.download_flow[task_index]
			finish_time := time.Now()
			dt.download_flow[task_index] = TaskRecord{
				chunck_id,
				node_id,
				exist_record.start_time, 
				finish_time, 
				exist_record.download_rate,
				"finish"}

			dt.updateRouteTable("finish", node_id, download_rate)
			dt.chunck_state_list[chunck_id] = STATE_FILLED
		}
	}
}

func (dt *DownloadTask) updateRouteTable(msg string, node_id int, download_rate int){
	if msg == "start"{
		dt.rt_manager.OnTaskStart(node_id)
	}else{
		dt.rt_manager.OnTaskFinish(node_id, download_rate)
	}
	// update by route_table_manager
}

// func (dt *DownloadTask) Run_b2(){
// 	go dt.CollectMsg()
// 	for{
// 		cur_time := time.Now()
// 		filled_chunck_count := dt.getFilledChunckCount()
// 		// fmt.Printf("filled: %v\n", filled_chunck_count)
// 		if filled_chunck_count >= dt.chunck_count{
// 			fmt.Printf("Download: %v x %v KBytes\n", filled_chunck_count, CHUNCK_SIZE)
// 			dt.saveDownloadFlow()
// 			// println("Download Finish")
// 			break
// 		}

// 		local_free_bandwidth := dt.getLocalFreeBandwidth()
// 		fmt.Printf("%02d:%02d                              speed: %v KB/s\n", cur_time.Minute(), cur_time.Second(), dt.bandwidth-local_free_bandwidth)
// 		if local_free_bandwidth < 20{
// 			time.Sleep(time.Duration(500)*time.Millisecond)
// 			continue
// 		}

// 		next_chunck_id, err := dt.getNextUnfilledChunckId()
// 		if err != nil{
// 			time.Sleep(time.Duration(500)*time.Millisecond)
// 			continue
// 		}

// 		// dt.startDownloadChunck(next_chunck_id, local_free_bandwidth)
// 		available_node_list := dt.getAvailableNodeList(next_chunck_id)

// 		// 对比测试，这里可以随机选择节点
// 		select_node_id := dt.rt_manager.SelectACONode(available_node_list)

// 		remote_free_bandwidth, download_delay := dt.getRemoteBandwidthAndDelay(select_node_id)

// 		var download_rate int
// 		if local_free_bandwidth > remote_free_bandwidth{
// 			download_rate = remote_free_bandwidth
// 		}else{
// 			download_rate = local_free_bandwidth
// 		}
// 		go StartDownloadTask(next_chunck_id, select_node_id, download_rate, download_delay, dt.msg_ch)

// 		time.Sleep(time.Duration(500)*time.Millisecond)
// 	}
// }

// 	// for ,{
// 	// 	go StartDownloadTask(chunck_id, node_id, dl_rate, dl_delay, ch.msg_ch)
// 	// }

// 	filled_chunck_count := 0

// 	for{
// 		if filled_chunck_count >= dt.chunck_count{
// 			println("Download Finish")
// 			break
// 		}

// 		if dt.free_banawidth < 20_000{
// 			time.Sleep(time.Duration(1)*time.Second)
// 			continue
// 		}

// 		next_chunck_id := dt.getNextUnfilledChunckId()
// 		available_node_list := dt.getAvailableNodeList(next_chunck_id)
// 		select_node_id := dt.rt_manager.SelectACORoute(available_node_list)

// 		remote_free_rate, download_delay := dt.getRemoteBandwidth(select_node_id)
// 		// local_free_rate := dt.getLocalFreeBandwidth()
// 		local_free_rate := dt.free_banawidth

// 		var download_rate int
// 		if local_free_rate > remote_free_rate{
// 			download_rate = remote_free_rate
// 		}else{
// 			download_rate = local_free_rate
// 		}
// 		// download_rate :=  
// 		go StartDownloadTask(chunck_id, node_id, download_rate, download_delay, dt.msg_ch)
// 	}
// }