package main

import (
	"mm_server/libs/log"
	"mm_server/proto/gen_go/client_message"
	"mm_server/proto/gen_go/server_message"
	"mm_server/src/login_server/login_db"

	"time"

	"github.com/golang/protobuf/proto"
)

func new_register_handler(account, password string, is_guest bool) (err_code int32, resp_data []byte) {
	if len(account) > 128 {
		log.Error("Account[%v] length %v too long", account, len(account))
		return -1, nil
	}

	if len(password) > 32 {
		log.Error("Account[%v] password[%v] length %v too long", account, password, len(password))
		return -1, nil
	}

	if account_record_mgr.Has(account) {
		log.Error("Account[%v] already exists", account)
		return int32(msg_client_message.E_ERR_ACCOUNT_ALREADY_REGISTERED), nil
	}

	if !is_guest {
		err_code = _check_register(account, password)
		if err_code < 0 {
			return
		}
	}

	uid := _generate_account_uuid(account)
	if uid == "" {
		err_code = -1
		return
	}

	new_record := login_db.Create_Account()
	new_record.Set_AccountId(account)
	new_record.Set_UniqueId(uid)
	new_record.Set_Password(password)
	new_record.Set_RegisterTime(uint32(time.Now().Unix()))
	if is_guest {
		new_record.Set_Channel("guest")
	}

	account_table.Insert(new_record)

	var response msg_client_message.S2CRegisterResponse = msg_client_message.S2CRegisterResponse{
		Account:  account,
		Password: password,
		IsGuest:  is_guest,
	}

	var err error
	resp_data, err = proto.Marshal(&response)
	if err != nil {
		err_code = int32(msg_client_message.E_ERR_INTERNAL)
		log.Error("login_handler marshal response error: %v", err.Error())
		return
	}

	log.Debug("Account[%v] password[%v] registered", account, password)

	err_code = 1
	return
}

func new_bind_new_account_handler(server_id int32, account, password, new_account, new_password, new_channel string) (err_code int32, resp_data []byte) {
	if len(new_account) > 128 {
		log.Error("Account[%v] length %v too long", new_account, len(new_account))
		return -1, nil
	}

	if new_channel != "facebook" && len(new_password) > 32 {
		log.Error("Account[%v] password[%v] length %v too long", new_account, new_password, len(new_password))
		return -1, nil
	}

	if account == new_account {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_NAME_MUST_DIFFRENT_TO_OLD)
		log.Error("Account %v can not bind same new account", account)
		return
	}

	acc_record := account_record_mgr.Get(account)
	if acc_record == nil {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_NOT_REGISTERED)
		log.Error("Account %v not registered, cant bind new account", account)
		return
	}

	ban_record := ban_mgr.Get(acc_record.Get_UniqueId())
	if ban_record != nil && ban_record.Get_StartTime() > 0 {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_BE_BANNED)
		log.Error("Account %v has been banned, cant login", account)
		return
	}

	channel := acc_record.Get_Channel()
	if channel != "guest" && channel != "facebook" {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_NOT_GUEST)
		log.Error("Account %v not guest and not facebook user", account)
		return
	}

	if channel == "facebook" {
		err_code = _verify_facebook_login(account, password)
		if err_code < 0 {
			return
		}
	}

	if channel != "facebook" && acc_record.Get_BindNewAccount() != "" {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_ALREADY_BIND)
		log.Error("Account %v already bind", account)
		return
	}

	if account_record_mgr.Has(new_account) {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_NEW_BIND_ALREADY_EXISTS)
		log.Error("New Account %v to bind already exists", new_account)
		return
	}

	if new_channel != "" {
		if new_channel == "facebook" {
			err_code = _verify_facebook_login(new_account, new_password)
			if err_code < 0 {
				return
			}
		} else {
			err_code = -1
			log.Error("Account %v bind a unsupported channel %v account %v", account, new_channel, new_account)
			return
		}
	} else {
		err_code = _check_register(new_account, new_password)
		if err_code < 0 {
			return
		}
	}

	acc_record.Set_BindNewAccount(new_account)
	account_table.UpdateWithFieldName(acc_record, []string{"BindNewAccount"})

	register_time := acc_record.Get_RegisterTime()
	uid := acc_record.Get_UniqueId()
	last_server_id := acc_record.Get_ServerId()
	// new bind account
	acc_record = login_db.Create_Account()
	if new_channel == "" {
		acc_record.Set_Password(new_password)
	}
	acc_record.Set_RegisterTime(register_time)
	acc_record.Set_UniqueId(uid)
	acc_record.Set_OldAccount(account)
	acc_record.Set_ServerId(last_server_id)
	account_table.Insert(acc_record)

	game_agent := game_agent_manager.GetAgentByID(server_id)
	if nil == game_agent {
		err_code = int32(msg_client_message.E_ERR_PLAYER_SELECT_SERVER_NOT_FOUND)
		log.Error("login_http_handler get hall_agent failed")
		return
	}

	req := &msg_server_message.L2GBindNewAccountRequest{
		UniqueId:   uid,
		Account:    account,
		NewAccount: new_account,
	}
	game_agent.Send(uint16(msg_server_message.MSGID_L2G_BIND_NEW_ACCOUNT_REQUEST), req)

	response := &msg_client_message.S2CGuestBindNewAccountResponse{
		Account:     account,
		NewAccount:  new_account,
		NewPassword: new_password,
		NewChannel:  new_channel,
	}

	var err error
	resp_data, err = proto.Marshal(response)
	if err != nil {
		err_code = int32(msg_client_message.E_ERR_INTERNAL)
		log.Error("login_handler marshal response error: %v", err.Error())
		return
	}

	log.Debug("Account[%v] bind new account[%v]", account, new_account)
	err_code = 1
	return
}

func new_login_handler(account, password, channel string) (err_code int32, resp_data []byte) {
	now_time := time.Now()
	var err error
	var is_new bool
	acc_record := account_record_mgr.Get(account)
	if config.VerifyAccount {
		if channel == "" {
			if acc_record == nil {
				err_code = int32(msg_client_message.E_ERR_PLAYER_ACC_OR_PASSWORD_ERROR)
				log.Error("Account %v not exist", account)
				return
			}
			if acc_record.Get_Password() != password {
				err_code = int32(msg_client_message.E_ERR_PLAYER_ACC_OR_PASSWORD_ERROR)
				log.Error("Account %v password %v invalid", account, password)
				return
			}
		} else if channel == "facebook" {
			err_code = _verify_facebook_login(account, password)
			if err_code < 0 {
				return
			}
			if acc_record == nil {
				acc_record = login_db.Create_Account()
				acc_record.Set_AccountId(account)
				acc_record.Set_Channel("facebook")
				acc_record.Set_RegisterTime(uint32(now_time.Unix()))
				is_new = true
			}
			acc_record.Set_Password(password)
		} else if channel == "guest" {
			if acc_record == nil {
				acc_record = login_db.Create_Account()
				acc_record.Set_Channel("guest")
				acc_record.Set_RegisterTime(uint32(now_time.Unix()))
				is_new = true
			} else {
				if acc_record.Get_Password() != password {
					err_code = int32(msg_client_message.E_ERR_PLAYER_ACC_OR_PASSWORD_ERROR)
					log.Error("Account %v password %v invalid", account, password)
					return
				}
			}
		} else {
			log.Error("Account %v use unsupported channel %v login", account, channel)
			return -1, nil
		}
	} else {
		if acc_record == nil {
			acc_record = login_db.Create_Account()
			acc_record.Set_AccountId(account)
			acc_record.Set_RegisterTime(uint32(now_time.Unix()))
			is_new = true
		}
	}

	if acc_record.Get_UniqueId() == "" {
		uid := _generate_account_uuid(account)
		if uid != "" {
			acc_record.Set_UniqueId(uid)
		}
	}

	ban_record := ban_mgr.Get(acc_record.Get_UniqueId())
	if ban_record != nil && ban_record.Get_StartTime() > 0 {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_BE_BANNED)
		log.Error("Account %v has been banned, cant login", account)
		return
	}

	// --------------------------------------------------------------------------------------------
	// 选择默认服
	server_id := acc_record.Get_ServerId()
	if server_id <= 0 {
		server := server_list.RandomOneServer()
		if server == nil {
			err_code = int32(msg_client_message.E_ERR_INTERNAL)
			log.Error("Server List random null !!!")
			return
		}
		server_id = uint32(server.Id)
		acc_record.Set_ServerId(server_id)
	}

	var hall_ip, token string
	err_code, hall_ip, token = _select_server(acc_record.Get_UniqueId(), account, int32(server_id))
	if err_code < 0 {
		return
	}
	// --------------------------------------------------------------------------------------------

	account_login(account, token, "")

	acc_record.Set_Token(token)

	if is_new {
		account_table.Insert(acc_record)
	} else {
		account_table.UpdateAll(acc_record)
	}

	response := &msg_client_message.S2CLoginResponse{
		Acc:    account,
		Token:  token,
		GameIP: hall_ip,
	}

	if channel == "guest" {
		response.BoundAccount = acc_record.Get_BindNewAccount()
	}

	resp_data, err = proto.Marshal(response)
	if err != nil {
		err_code = int32(msg_client_message.E_ERR_INTERNAL)
		log.Error("login_handler marshal response error: %v", err.Error())
		return
	}

	log.Debug("Account[%v] channel[%v] logined", account, channel)

	return
}

func new_set_password_handler(account, password, new_password string) (err_code int32, resp_data []byte) {
	acc_record := account_record_mgr.Get(account)
	if acc_record == nil {
		err_code = int32(msg_client_message.E_ERR_PLAYER_NOT_EXIST)
		log.Error("set_password_handler account[%v] not found", account)
		return
	}
	ban_record := ban_mgr.Get(acc_record.Get_UniqueId())
	if ban_record != nil && ban_record.Get_StartTime() > 0 {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_BE_BANNED)
		log.Error("Account %v has been banned, cant login", account)
		return
	}
	if acc_record.Get_Password() != password {
		err_code = int32(msg_client_message.E_ERR_ACCOUNT_PASSWORD_INVALID)
		log.Error("set_password_handler account[%v] password is invalid", account)
		return
	}
	acc_record.Set_Password(new_password)
	account_table.UpdateWithFieldName(acc_record, []string{"Password"})

	response := &msg_client_message.S2CSetLoginPasswordResponse{
		Account:     account,
		Password:    password,
		NewPassword: new_password,
	}

	var err error
	resp_data, err = proto.Marshal(response)
	if err != nil {
		err_code = int32(msg_client_message.E_ERR_INTERNAL)
		log.Error("set_password_handler marshal response error: %v", err.Error())
		return
	}

	return
}
