package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (p Params) GetInt64(key string) (number int64, err error) {
	err = json.Unmarshal(p[key], &number)
	return
}

func (p Params) GetString(key string) (str string, err error) {
	err = json.Unmarshal(p[key], &str)
	return
}

func (p Params) GetTime(key string) (t time.Time, err error) {
	var timestamp int64
	err = json.Unmarshal(p[key], &timestamp)
	if err != nil {
		return
	}
	t = time.Unix(timestamp, 0)
	return
}

func reportError(err error) {
	if err != nil {
		err_msg := err.Error()
		log.Println(err_msg)
		sendPrivateMessage(config.SuperUser.QQNumber, err_msg)
	}
}

func json_trans(json string) string {
	json = strings.Replace(json, "\\", "\\\\", -1)
	json = strings.Replace(json, "\"", "\\\"", -1)
	json = strings.Replace(json, "\n", "\\n", -1)
	json = strings.Replace(json, "\r", "\\r", -1)
	json = strings.Replace(json, "\t", "\\t", -1)
	return json
}

func GetGroups(loginQQ int64, cookies string, csrf_token int64) (groups map[int64]Group) {
	url_addr := "http://qun.qzone.qq.com/cgi-bin/get_group_list?groupcount=4&count=4&callbackFun=_GetGroupPortal&uin=" + strconv.FormatInt(loginQQ, 10) + "&g_tk=" + strconv.FormatInt(csrf_token, 10) + "&ua=Mozilla%2F5.0%20"
	//log.Println(url_addr)
	//log.Println(cookies)

	// proxy, err := url.Parse("http://10.30.3.16:7777")
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	http_client := &http.Client{
		Transport: &http.Transport{
			// Proxy:             http.ProxyURL(proxy),
			DisableKeepAlives: true,
		},
	}

	r, err := http.NewRequest("GET", url_addr, nil)
	if err != nil {
		log.Println(err)
		return
	}
	r.Header.Set("Cookie", cookies)
	resp, err := http_client.Do(r)
	if err != nil {
		log.Println(err)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	resp.Body.Close()
	groups_resp := string(b)
	if strings.HasPrefix(groups_resp, "_GetGroupPortal_Callback(") && strings.HasSuffix(groups_resp, ");") {
		groups_resp = groups_resp[25 : len(groups_resp)-2]
		groupsjson := GetGroupsJson{}
		err = json.Unmarshal([]byte(groups_resp), &groupsjson)
		if err == nil {
			for _, g := range groupsjson.Data.Group {
				row := db.QueryRow("select id, name from groups where group_number = $1", g.Groupid)
				var groupname string
				var group_id int64
				err = row.Scan(&group_id, &groupname)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
					}
					_, err = trans.Exec("insert into groups(group_number, name) values($1, $2)", g.Groupid, g.Groupname)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				} else if err != nil {
					reportError(err)
				} else if groupname != g.Groupname {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
					}
					_, err = trans.Exec("update groups set name = $1 where group_number = $2", g.Groupname, g.Groupid)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				}
			}
		} else {
			reportError(err)
			return
		}

		rows, err := db.Query("select id, group_number, name from groups")
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		if groups == nil {
			groups = make(map[int64]Group)
		}
		i := 0
		for rows.Next() {
			var groupname string
			var group_id int64
			var groupno int64
			rows.Scan(&group_id, &groupno, &groupname)
			group := Group{group_id, groupno, groupname}
			groups[groupno] = group
			log.Printf("%d: %s[%d]", i, group.GroupName, groupno)
			i++
		}
	}
	return
}

func GetGroupMembers(group Group, loginQQ int64, cookies string, csrf_token int64) (nicknames_in_group map[int64]Member) {
	nicknames_in_group = make(map[int64]Member, 100)

	url_addr := fmt.Sprintf("http://qun.qzone.qq.com/cgi-bin/get_group_member?callbackFun=_GroupMember&uin=%d&groupid=%d&neednum=1&r=0.5421284231954122&g_tk=%d&ua=Mozilla%2F4.0%20(compatible%3B%20MSIE%207.0%3B%20Windows%20NT%205.1%3B%20Trident%2F4.0)", loginQQ, group.GroupNo, csrf_token)
	//log.Println(url_addr)
	//log.Println(cookies)

	// proxy, err := url.Parse("http://10.30.3.16:7777")
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	http_client := &http.Client{
		Transport: &http.Transport{
			// Proxy:             http.ProxyURL(proxy),
			DisableKeepAlives: true,
		},
	}

	r, err := http.NewRequest("GET", url_addr, nil)
	if err != nil {
		log.Println(err)
		return
	}
	r.Header.Set("Cookie", cookies)
	resp, err := http_client.Do(r)
	if err != nil {
		log.Println(err)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	resp.Body.Close()
	group_members_resp := string(b)
	if strings.HasPrefix(group_members_resp, "_GroupMember_Callback(") && strings.HasSuffix(group_members_resp, ");") {
		group_members_resp = group_members_resp[22 : len(group_members_resp)-2]
		memberjson := GetGroupMembersJson{}
		err = json.Unmarshal([]byte(group_members_resp), &memberjson)
		if err == nil {
			for _, m := range memberjson.Data.Item {
				var userid, memberid int64
				userid, err = get_user_id(m.Uin)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					err = trans.QueryRow("insert into users(qq_number) values($1) returning id", m.Uin).Scan(&userid)
					if err != nil {
						log.Println(err)
						trans.Rollback()
						continue
					} else {
						trans.Commit()
					}
				} else if err != nil {
					log.Println(err)
					continue
				}
				user := User{userid, m.Uin, ""}

				row := db.QueryRow("SELECT id, nickname, rights FROM group_members where group_id = $1 and user_id = $2", group.Id, user.Id)
				var nickname string
				var rights int
				err = row.Scan(&memberid, &nickname, &rights)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					err = trans.QueryRow("insert into group_members(group_id, user_id, nickname) values($1, $2, $3)  returning id", group.Id, user.Id, m.Nick).Scan(&memberid)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				} else if err != nil {
					reportError(err)
				} else if nickname != m.Nick {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					_, err = trans.Exec("update group_members set nickname = $1 where group_id = $2 and user_id = $3", m.Nick, group.Id, user.Id)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				}
			}
		} else {
			reportError(err)
			return
		}

		rows, err := db.Query("SELECT n.id, n.user_id, u.qq_number,n.nickname,n.rights FROM group_members n, users u where n.user_id = u.id and n.group_id=$1", group.Id)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			var nickname string
			var userqq, user_id, member_id int64
			var rights int
			err = rows.Scan(&member_id, &user_id, &userqq, &nickname, &rights)
			if err != nil {
				panic(err)
			}
			nicknames_in_group[userqq] = Member{member_id, user_id, group.Id, nickname, rights}
		}
	}
	return
}

func spliteN(s string) (result []string) {
	max := 100
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) < 2 || len(r) > 30 {
		return result
	}
	var c = 0
	for i := 0; i < len(r); i++ {
		for j := 0; j+i <= len(r) && j < i; j++ {
			flag := false
			for idx, c := range r[j:] {
				if c == '[' {
					flag = true
					continue
				}
				if c == ']' {
					flag = false
					continue
				}
				if flag {
					continue
				}
				if idx >= i && !flag {
					i = idx
					break
				}
			}
			key := []rune(strings.TrimSpace(string(r[j:i])))
			if len(key) >= 2 && len(key) < 5 {
				contain := false
				for _, item := range result {
					if item == string(key) {
						contain = true
						break
					}
				}
				if !contain {
					result = append(result, string(key))
					c++
					if c > max {
						return
					}
				}
			}
		}
	}
	return result
}

func message_length(message string) (lenght int) {
	flag := false
	for _, c := range message {
		if c == '[' || (flag && c != ']') {
			flag = true
			continue
		}
		if c == ']' {
			flag = false
		}
		lenght++
	}
	return
}

func get_group_id(groupnum int64) (groupid int64, err error) {
	if group, ok := groups[groupnum]; ok {
		groupid = group.Id
		return
	}
	row := db.QueryRow("select id, name from groups where group_number = $1", groupnum)
	var groupName string
	err = row.Scan(&groupid, &groupName)
	g := Group{groupid, groupnum, groupName}
	if groups == nil {
		groups = make(map[int64]Group)
	}
	groups[groupnum] = g
	return
}

func get_user_id(qqnum int64) (userid int64, err error) {

	if user, ok := users[qqnum]; ok {
		userid = user.Id
		return
	}
	row := db.QueryRow("select id, qq_name from users where qq_number = $1", qqnum)
	var name sql.NullString
	err = row.Scan(&userid, &name)
	if users == nil {
		users = make(map[int64]User)
	}
	if name.Valid {
		users[qqnum] = User{userid, qqnum, name.String}
	} else {
		users[qqnum] = User{userid, qqnum, ""}
	}
	return
}
