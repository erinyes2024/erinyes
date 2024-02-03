package parser

import (
	"erinyes/conf"
	"erinyes/logs"
)

type SysdigParser struct {
	execveMap map[string]map[string]string
	pusher    *Pusher
}

func NewSysdigParser(pusher *Pusher) *SysdigParser {
	return &SysdigParser{
		pusher:    pusher,
		execveMap: make(map[string]map[string]string),
	}
}

func (p *SysdigParser) ParserType() string {
	return SYSDIG
}

func (p *SysdigParser) ParsePushLine(rawLine string) error {
	err, sysdigLog := SplitSysdigLine(rawLine)
	if err != nil {
		return err
	}
	pl := ParsedLog{}

	if sysdigLog.IsProcessCall() {
		if sysdigLog.EventType == SYS_EXECVE {
			key := sysdigLog.HostID + "#" + sysdigLog.ContainerID + "#" + sysdigLog.VPid
			if sysdigLog.Dir == ">" {
				p.execveMap[key] = make(map[string]string)
				p.execveMap[key]["process_name"] = sysdigLog.ProcessName
				p.execveMap[key]["process_exepath"] = sysdigLog.Cmd
				return nil
			}
			if value, exists := p.execveMap[key]; exists {
				// 1. process ->(execve) process
				pl.StartVertex = ProcessVertex{
					HostID:         sysdigLog.HostID,
					HostName:       sysdigLog.HostName,
					ContainerID:    sysdigLog.ContainerID,
					ContainerName:  sysdigLog.ContainerName,
					ProcessVPID:    sysdigLog.VPid,
					ProcessName:    value["process_name"],
					ProcessExepath: value["process_exepath"],
				}
				pl.EndVertex = ProcessVertex{
					HostID:         sysdigLog.HostID,
					HostName:       sysdigLog.HostName,
					ContainerID:    sysdigLog.ContainerID,
					ContainerName:  sysdigLog.ContainerName,
					ProcessVPID:    sysdigLog.VPid,
					ProcessName:    sysdigLog.ProcessName,
					ProcessExepath: sysdigLog.Cmd,
				}
				pl.Log = ParsedSysdigLog{
					EventCLass: PROCESS,
					Relation:   SYS_EXECVE,
					Operation:  SYS_EXECVE,
					Time:       sysdigLog.Time,
					UUID:       sysdigLog.GetLastRequestUUID(),
				}
				delete(p.execveMap, key)
			}
		} else { // clone fork vfork
			if sysdigLog.Dir == ">" || sysdigLog.Ret == "0" || sysdigLog.Ret == "-1" {
				return nil
			}
			if sysdigLog.Ret == NASTR {
				logs.Logger.Warnf("Process create but ret nil")
				return nil
			}
			// 2. process ->(fork vfork clone) process
			pl.StartVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.EndVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.Ret,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: PROCESS,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		}
	} else if sysdigLog.IsNetCall() {
		if sysdigLog.Dir == ">" {
			return nil
		}
		if sysdigLog.EventType == SYS_SENDTO || sysdigLog.EventType == SYS_CONNECT || sysdigLog.EventType == SYS_WRITE {
			if sysdigLog.Fd == NASTR || sysdigLog.Fd == NILSTR || !IsSocket(sysdigLog.Fd) {
				return nil
			}
			_, _, dstIP, dstPort := sysdigLog.MustExtractFourTuple()
			// 3. process ->(sendto connect write) socket
			pl.StartVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			if dstIP == "localhost" {
				pl.EndVertex = SocketVertex{
					HostID:        sysdigLog.HostID,
					HostName:      sysdigLog.HostName,
					ContainerID:   sysdigLog.ContainerID,
					ContainerName: sysdigLog.ContainerName,
					DstIP:         dstIP,
					DstPort:       dstPort,
				}
			} else {
				pl.EndVertex = SocketVertex{
					HostID:        conf.MockHostID,
					HostName:      conf.MockHostName,
					ContainerID:   conf.OuterContainerID,
					ContainerName: conf.OuterContainerName,
					DstIP:         dstIP,
					DstPort:       dstPort,
				}
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: NETWORKV1,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		} else if sysdigLog.EventType == SYS_RECVFROM || sysdigLog.EventType == SYS_READ {
			if sysdigLog.Fd == NASTR || sysdigLog.Fd == NILSTR || !IsSocket(sysdigLog.Fd) {
				return nil
			}
			_, _, dstIP, dstPort := sysdigLog.MustExtractFourTuple()
			// 4. socket ->(recvfrom read) process
			if dstIP == "localhost" {
				pl.StartVertex = SocketVertex{
					HostID:        sysdigLog.HostID,
					HostName:      sysdigLog.HostName,
					ContainerID:   sysdigLog.ContainerID,
					ContainerName: sysdigLog.ContainerName,
					DstIP:         dstIP,
					DstPort:       dstPort,
				}
			} else {
				pl.StartVertex = SocketVertex{
					HostID:        conf.MockHostID,
					HostName:      conf.MockHostName,
					ContainerID:   conf.OuterContainerID,
					ContainerName: conf.OuterContainerName,
					DstIP:         dstIP,
					DstPort:       dstPort,
				}
			}
			pl.EndVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: NETWORKV2,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		} else if sysdigLog.EventType == SYS_BIND || sysdigLog.EventType == SYS_LISTEN {
			port, valid := sysdigLog.ExtractPort()
			if !valid {
				return nil
			}
			// 5. process ->(bind listen) socket
			pl.StartVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.EndVertex = SocketVertex{
				HostID:        sysdigLog.HostID,
				HostName:      sysdigLog.HostName,
				ContainerID:   sysdigLog.ContainerID,
				ContainerName: sysdigLog.ContainerName,
				DstIP:         "localhost",
				DstPort:       port,
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: NETWORKV1,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		} else if sysdigLog.EventType == SYS_ACCEPT || sysdigLog.EventType == SYS_ACCEPT4 { // ignore
			return nil
		} else {
			logs.Logger.Errorf("Unkown event type %s", sysdigLog.EventType)
			return nil
		}
	} else if sysdigLog.IsFileCall() {
		if sysdigLog.Dir == ">" {
			return nil
		}
		if sysdigLog.IsNodeTriggerStartLog() {
			conf.NodeLastRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID] = sysdigLog.Info[3]
			return nil
		} else if sysdigLog.IsNodeTriggerEndLog() {
			if conf.NodeLastRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID] == sysdigLog.Info[3] {
				conf.NodeLastRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID] = UNKNOWN
			}
			return nil
		} else if sysdigLog.IsOfwatchdogTriggerStartLog() {
			if _, ok := conf.OfwatchdogRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID]; !ok {
				conf.OfwatchdogRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID] = make(map[string]bool)
			}
			conf.OfwatchdogRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID][sysdigLog.Info[3]] = true
			return nil
		} else if sysdigLog.IsOfwatchdogTriggerEndLog() {
			if _, ok := conf.OfwatchdogRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID]; !ok {
				return nil
			}
			delete(conf.OfwatchdogRequestUUIDMap[sysdigLog.HostID+"#"+sysdigLog.ContainerID], sysdigLog.Info[3])
			return nil
		}

		if sysdigLog.Fd == NASTR || sysdigLog.Fd == NILSTR {
			return nil
		}
		//logs.Logger.Infof("[Debug] file: %s", sysdigLog.Fd)
		if sysdigLog.EventType == SYS_WRITE || sysdigLog.EventType == SYS_WRITEV {
			// 6. process ->(write writev) file
			pl.StartVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.EndVertex = FileVertex{
				HostID:        sysdigLog.HostID,
				HostName:      sysdigLog.HostName,
				ContainerID:   sysdigLog.ContainerID,
				ContainerName: sysdigLog.ContainerName,
				FilePath:      sysdigLog.FilteredFilePath(),
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: FILEV1,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		} else if sysdigLog.EventType == SYS_READ || sysdigLog.EventType == SYS_READV {
			// 7. file ->(read readv) process
			pl.StartVertex = FileVertex{
				HostID:        sysdigLog.HostID,
				HostName:      sysdigLog.HostName,
				ContainerID:   sysdigLog.ContainerID,
				ContainerName: sysdigLog.ContainerName,
				FilePath:      sysdigLog.FilteredFilePath(),
			}
			pl.EndVertex = ProcessVertex{
				HostID:         sysdigLog.HostID,
				HostName:       sysdigLog.HostName,
				ContainerID:    sysdigLog.ContainerID,
				ContainerName:  sysdigLog.ContainerName,
				ProcessVPID:    sysdigLog.VPid,
				ProcessName:    sysdigLog.ProcessName,
				ProcessExepath: sysdigLog.Cmd,
			}
			pl.Log = ParsedSysdigLog{
				EventCLass: FILEV2,
				Relation:   sysdigLog.EventType,
				Operation:  sysdigLog.EventType,
				Time:       sysdigLog.Time,
				UUID:       sysdigLog.GetLastRequestUUID(),
			}
		} else if sysdigLog.EventType == SYS_OPEN || sysdigLog.EventType == SYS_OPENAT {
			err, kind := sysdigLog.ConvertSysOpen()
			if err != nil {
				logs.Logger.WithError(err).Errorf("Parse open syscall failed")
				return nil
			}
			if kind == SYS_READ { // file ->(read) process
				pl.StartVertex = FileVertex{
					HostID:        sysdigLog.HostID,
					HostName:      sysdigLog.HostName,
					ContainerID:   sysdigLog.ContainerID,
					ContainerName: sysdigLog.ContainerName,
					FilePath:      sysdigLog.FilteredFilePath(),
				}
				pl.EndVertex = ProcessVertex{
					HostID:         sysdigLog.HostID,
					HostName:       sysdigLog.HostName,
					ContainerID:    sysdigLog.ContainerID,
					ContainerName:  sysdigLog.ContainerName,
					ProcessVPID:    sysdigLog.VPid,
					ProcessName:    sysdigLog.ProcessName,
					ProcessExepath: sysdigLog.Cmd,
				}
				pl.Log = ParsedSysdigLog{
					EventCLass: FILEV2,
					Relation:   SYS_READ,
					Operation:  SYS_READ,
					Time:       sysdigLog.Time,
					UUID:       sysdigLog.GetLastRequestUUID(),
				}
			} else if kind == SYS_WRITE { // process ->(write) file
				pl.StartVertex = ProcessVertex{
					HostID:         sysdigLog.HostID,
					HostName:       sysdigLog.HostName,
					ContainerID:    sysdigLog.ContainerID,
					ContainerName:  sysdigLog.ContainerName,
					ProcessVPID:    sysdigLog.VPid,
					ProcessName:    sysdigLog.ProcessName,
					ProcessExepath: sysdigLog.Cmd,
				}
				pl.EndVertex = FileVertex{
					HostID:        sysdigLog.HostID,
					HostName:      sysdigLog.HostName,
					ContainerID:   sysdigLog.ContainerID,
					ContainerName: sysdigLog.ContainerName,
					FilePath:      sysdigLog.FilteredFilePath(),
				}
				pl.Log = ParsedSysdigLog{
					EventCLass: FILEV1,
					Relation:   SYS_WRITE,
					Operation:  SYS_WRITE,
					Time:       sysdigLog.Time,
					UUID:       sysdigLog.GetLastRequestUUID(),
				}
			}
		} else {
			logs.Logger.Errorf("Unkown event type %s", sysdigLog.EventType)
			return nil
		}
	} else {
		// do nothing, just ignore it
		// In fact, other event type should be filtered when collecting
		logs.Logger.Errorf("unknown syscall type is %s", sysdigLog.EventType)
		return nil
	}
	p.pusher.PushParsedLog(pl)
	return nil
}
