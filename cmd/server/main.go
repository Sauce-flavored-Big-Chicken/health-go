package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

const (
	defaultHTTPURL = "http://121.37.25.126/community/public"
	maxUploadSize  = 40 * 1024 * 1024
)

type app struct {
	db      *sql.DB
	httpURL string
	jwtKey  string
}

type tokenResult struct {
	Code int
	Msg  string
	UID  string
}

func main() {
	loadDotEnv(".env")

	dbDriver := strings.ToLower(getEnv("DB_DRIVER", "sqlite"))
	dsn := getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/health?charset=utf8&parseTime=true&loc=Local")
	if dbDriver == "sqlite" {
		dsn = getEnv("SQLITE_PATH", "health.db")
	}
	httpURL := getEnv("HTTP_URL", defaultHTTPURL)
	addr := getEnv("APP_ADDR", ":8080")

	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}

	sum := md5.Sum([]byte("tp6.1.4"))
	a := &app{db: db, httpURL: httpURL, jwtKey: hex.EncodeToString(sum[:])}
	if err = ensureStaticAssets(); err != nil {
		panic(err)
	}
	if err = a.ensureSupplementalTables(); err != nil {
		panic(err)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	registerRoutes(r, a)

	if err = r.Run(addr); err != nil {
		panic(err)
	}
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

func registerRoutes(r *gin.Engine, a *app) {
	r.POST("/prod-api/api/login", a.login)
	r.GET("/prod-api/api/SMSCode", a.smsCode)
	r.POST("/prod-api/api/phone/login", a.phoneLogin)
	r.POST("/prod-api/api/register", a.register)
	r.POST("/logout", a.logout)

	auth := r.Group("/")
	auth.Use(a.authMiddleware())

	auth.GET("/prod-api/api/user/getUserInfo", a.getUserInfo)
	auth.PUT("/prod-api/api/user/updateUserInfo", a.updateUserInfo)
	auth.PUT("/prod-api/api/user/resetPwd", a.resetPwd)
	auth.PUT("/prod-api/api/user/resetPwdByUserName", a.resetPwdByUserName)
	auth.PUT("/prod-api/api/user/resetName", a.resetName)
	auth.GET("/prod-api/api/user/avatarList", a.getAvatarList)
	auth.PUT("/prod-api/api/user/updateUserAvatar", a.updateUserAvatar)
	auth.GET("/prod-api/api/user/getContactInfo", a.getContactInfo)
	auth.PUT("/prod-api/api/user/updateContactInfo", a.updateContactInfo)

	auth.GET("/prod-api/api/rotation/list", a.getBannerList)

	auth.GET("/prod-api/api/notice/list", a.getNoticeList)
	auth.GET("/prod-api/api/notice/:id", a.getNoticeInfo)
	auth.PUT("/prod-api/api/readNotice/:id", a.readNotice)

		auth.GET("/prod-api/api/community/list", a.getCommunityList)
		auth.GET("/prod-api/api/community/dynamic/list", a.getCommunityDynamicList)
		auth.GET("/prod-api/api/friendly_neighborhood/list", a.getFriendlyNeighborhoodList)
		auth.POST("/prod-api/api/friendly_neighborhood/add", a.addFriendlyNeighborhoodPost)
		auth.POST("/prod-api/api/friendly_neighborhood/add/comment", a.addFriendlyNeighborhoodComment)
		auth.GET("/prod-api/api/friendly_neighborhood/:id", a.getFriendlyNeighborhoodInfo)

		auth.GET("/prod-api/api/press/category/list", a.getCategoryList)
	auth.GET("/prod-api/api/press/category/newsList", a.getNewsList)
	auth.GET("/prod-api/api/press/newsList", a.getNewsAllList)
	auth.GET("/prod-api/api/press/news/:id", a.getNewsInfo)
	auth.PUT("/prod-api/api/press/like", a.likeNews)
	auth.PUT("/prod-api/api/press/like/:id", a.likeNews)

	auth.POST("/prod-api/api/comment/pressComment", a.addComment)
	auth.GET("/prod-api/api/comment/comment/:id", a.getCommentList)
	auth.PUT("/prod-api/api/comment/like/:id", a.likeComment)

	auth.POST("/prod-api/api/common/upload", a.upload)

	auth.GET("/prod-api/api/activity/topList", a.getActivityTopList)
	auth.GET("/prod-api/api/activity/list", a.getActivityList)
	auth.GET("/prod-api/api/activity/List", a.getActivityList)
	auth.GET("/prod-api/api/activity/category/list/:id", a.getActivityCategoryList)
	auth.GET("/prod-api/api/activity/:id", a.getActivityInfo)
		auth.POST("/prod-api/api/activity/search", a.searchActivityList)
		auth.POST("/prod-api/api/registration", a.registerActivity)
		auth.PUT("/prod-api/api/checkin/:id", a.checkInActivity)
		auth.PUT("/prod-api/api/registration/comment/:id", a.commentActivity)

	auth.GET("/prod-api/api/course/courseList", a.getCourseList)
	auth.GET("/prod-api/api/course/course/:id", a.getCourseInfo)

	auth.GET("/api/material/getMaterialInfo/:moduleId", a.getMaterialInfo)
	auth.GET("/api/answer/getInfo/:moduleId", a.getAnswerInfo)
	auth.POST("/api/answer/upload/:moduleId", a.uploadAnswerMaterial)
	auth.DELETE("/api/answer/delMaterial/:id", a.delMaterial)

	r.Static("/static", "public/static")
	r.Static("/storage", "public/storage")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Max-Age", "1800")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-CSRF-TOKEN, X-Requested-With, Token")
		if strings.ToUpper(c.Request.Method) == http.MethodOptions {
			c.Status(http.StatusOK)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *app) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			respond(c, map[string]any{"code": 401, "msg": "缺少 Authorization，请先登录后再访问"})
			c.Abort()
			return
		}
		res := a.checkToken(token)
		if res.Code != 200 {
			respond(c, map[string]any{
				"code": 401,
				"msg":  res.Msg,
				"data": "请求访问: " + strings.TrimPrefix(c.Request.URL.Path, "/") + ", 认证失败, 无法访问系统资源",
			})
			c.Abort()
			return
		}
		c.Set("uid", res.UID)
		c.Next()
	}
}

func (a *app) generateToken(uid string) string {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss":  a.jwtKey,
		"aud":  "",
		"iat":  now,
		"nbf":  now,
		"exp":  now + 7200,
		"data": map[string]any{"uid": uid},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := t.SignedString([]byte(a.jwtKey))
	return token
}

func (a *app) checkToken(token string) tokenResult {
	res := tokenResult{Code: 404}
	claims := jwt.MapClaims{}
	parser := jwt.NewParser(jwt.WithLeeway(60 * time.Second))
	parsed, err := parser.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return []byte(a.jwtKey), nil
	})
	if err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "signature"):
				res.Msg = "token签名不正确，请重新登录"
		case strings.Contains(msg, "expired"):
				res.Msg = "token已过期，请重新登录"
		case strings.Contains(msg, "not valid yet"):
				res.Msg = "token尚未生效或已失效，请重新登录"
		default:
				res.Msg = "token解析失败，请重新登录"
		}
		return res
	}
	if !parsed.Valid {
			res.Msg = "token校验失败，请重新登录"
		return res
	}
	data, ok := claims["data"].(map[string]any)
	if !ok {
			res.Msg = "token数据格式错误，请重新登录"
		return res
	}
	uid, _ := data["uid"].(string)
	res.Code = 200
	res.UID = uid
	return res
}

func (a *app) login(c *gin.Context) {
	userName := getParam(c, "userName")
	passWord := getParam(c, "passWord")
	if userName == "" || passWord == "" {
		respond(c, map[string]any{"code": 400, "msg": "用户名和密码不能为空"})
		return
	}

	row, err := a.queryOne(`SELECT id,passWord FROM tp_user WHERE userName=? AND status=1`, userName)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "登录失败，请稍后重试"})
		return
	}
	if row == nil {
		respond(c, map[string]any{"code": 400, "msg": "用户名不存在"})
		return
	}
	if toString(row["passWord"]) != passWord {
		respond(c, map[string]any{"code": 400, "msg": "密码错误"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "token": a.generateToken(userName)})
}

	func (a *app) smsCode(c *gin.Context) {
		phone := getParam(c, "phone")
		if phone == "" {
			respond(c, map[string]any{"code": 400, "msg": "手机号不能为空"})
			return
		}
		randCode, err := generateSMSCode()
	if err != nil {
		respond(c, map[string]any{"code": 500, "msg": "验证码生成失败"})
		return
	}

	affected, err := a.exec(`UPDATE tp_user SET SMSCode=? WHERE phonenumber=?`, randCode, phone)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "未找到该手机号对应的用户"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": randCode})
}

func (a *app) phoneLogin(c *gin.Context) {
	phone := getParam(c, "phone")
	smsCode := getParam(c, "SMSCode")
	if phone == "" || smsCode == "" {
		respond(c, map[string]any{"code": 400, "msg": "手机号和验证码不能为空"})
		return
	}
	row, err := a.queryOne(`SELECT id,SMSCode FROM tp_user WHERE phonenumber=? AND status=1`, phone)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "登录失败，请稍后重试"})
		return
	}
	if row == nil {
		respond(c, map[string]any{"code": 400, "msg": "手机号不存在"})
		return
	}
	if toString(row["SMSCode"]) != smsCode {
		respond(c, map[string]any{"code": 400, "msg": "验证码错误"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "token": a.generateToken(phone)})
}

func (a *app) logout(c *gin.Context) {
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) register(c *gin.Context) {
	now := time.Now().Unix()
	if getParam(c, "userName") == "" || getParam(c, "passWord") == "" || getParam(c, "phonenumber") == "" || getParam(c, "sex") == "" {
		respond(c, map[string]any{"code": 400, "msg": "缺少必填字段：userName、passWord、phonenumber、sex"})
		return
	}
	_, err := a.exec(`INSERT INTO tp_user (userName,nickName,passWord,avatar,phonenumber,sex,email,idCard,address,introduction,createTime)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		getParam(c, "userName"), getParam(c, "nickName"), getParam(c, "passWord"), getParam(c, "avatar"),
		getParam(c, "phonenumber"), getParam(c, "sex"), getParam(c, "email"), getParam(c, "idCard"),
		getParam(c, "address"), getParam(c, "introduction"), now)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "注册失败，用户名或手机号可能已存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getUserInfo(c *gin.Context) {
	uid := mustUID(c)
	row, err := a.getUserByUID(uid)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "未找到当前登录用户信息"})
		return
	}
	row["userId"] = row["id"]
	row["balance"] = row["money"]
	row["score"] = row["points"]
	row["avatar"] = avatarURL(a.baseURL(c), toString(row["avatar"]))
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) updateUserInfo(c *gin.Context) {
	uid := mustUID(c)
	fields := []struct {
		param  string
		column string
	}{
		{param: "nickName", column: "nickName"},
		{param: "avatar", column: "avatar"},
		{param: "phonenumber", column: "phonenumber"},
		{param: "sex", column: "sex"},
		{param: "email", column: "email"},
		{param: "idCard", column: "idCard"},
		{param: "address", column: "address"},
		{param: "introduction", column: "introduction"},
	}

	setClauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+3)
	for _, field := range fields {
		if !hasParam(c, field.param) {
			continue
		}
		setClauses = append(setClauses, field.column+"=?")
		args = append(args, getParam(c, field.param))
	}
	if len(setClauses) == 0 {
		respond(c, map[string]any{"code": 400, "msg": "没有可更新的用户字段"})
		return
	}

	setClauses = append(setClauses, "updateTime=?")
	args = append(args, time.Now().Unix(), uid, uid)

	sqlText := fmt.Sprintf("UPDATE tp_user SET %s WHERE status=1 AND (userName=? OR phonenumber=?)", strings.Join(setClauses, ","))
	affected, err := a.exec(sqlText, args...)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "更新个人信息失败，请确认用户存在且参数有效"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) resetPwd(c *gin.Context) {
	uid := mustUID(c)
	oldPassword := getParam(c, "oldPassword")
	newPassword := getParam(c, "newPassword")
	row, err := a.queryOne(`SELECT passWord FROM tp_user WHERE status=1 AND (userName=? OR phonenumber=?)`, uid, uid)
	if err != nil || row == nil || toString(row["passWord"]) != oldPassword {
		respond(c, map[string]any{"code": 400, "msg": "旧密码不正确"})
		return
	}
	affected, err := a.exec(`UPDATE tp_user SET passWord=?,updateTime=? WHERE status=1 AND (userName=? OR phonenumber=?)`,
		newPassword, time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "修改密码失败，请稍后重试"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) resetPwdByUserName(c *gin.Context) {
	userName := getParam(c, "userName")
	newPassword := getParam(c, "newPassword")
	if userName == "" || newPassword == "" {
		respond(c, map[string]any{"code": 400, "msg": "userName 和 newPassword 不能为空"})
		return
	}

	affected, err := a.exec(`UPDATE tp_user SET passWord=?,updateTime=? WHERE status=1 AND userName=?`,
		newPassword, time.Now().Unix(), userName)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "未找到要修改密码的用户名"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) resetName(c *gin.Context) {
	uid := mustUID(c)
	affected, err := a.exec(`UPDATE tp_user SET nickName=?,updateTime=? WHERE status=1 AND (userName=? OR phonenumber=?)`,
		getParam(c, "newName"), time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "修改昵称失败，请确认当前用户存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getAvatarList(c *gin.Context) {
	avatar := []map[string]any{
		{"id": 1, "avatar": "avatar1.png", "avatarUrl": "/static/avatar/avatar1.png"},
		{"id": 2, "avatar": "avatar2.png", "avatarUrl": "/static/avatar/avatar2.png"},
		{"id": 3, "avatar": "avatar3.png", "avatarUrl": "/static/avatar/avatar3.png"},
		{"id": 4, "avatar": "avatar4.png", "avatarUrl": "/static/avatar/avatar4.png"},
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": avatar, "total": 4})
}

func (a *app) updateUserAvatar(c *gin.Context) {
	uid := mustUID(c)
	affected, err := a.exec(`UPDATE tp_user SET avatar=?,updateTime=? WHERE status=1 AND (userName=? OR phonenumber=?)`,
		getParam(c, "avatar"), time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "更新头像失败，请确认当前用户存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getContactInfo(c *gin.Context) {
	uid := mustUID(c)
	id, err := a.getUserID(uid)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "未找到当前登录用户"})
		return
	}
	row, err := a.queryOne(`SELECT relationship,telephone,alternatePhone,createTime FROM tp_contact WHERE status=1 AND uId=?`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "未找到联系人信息"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) updateContactInfo(c *gin.Context) {
	uid := mustUID(c)
	id, err := a.getUserID(uid)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "未找到当前登录用户"})
		return
	}
	fields := []struct {
		param  string
		column string
	}{
		{param: "relationship", column: "relationship"},
		{param: "telephone", column: "telephone"},
		{param: "alternatePhone", column: "alternatePhone"},
	}
	setClauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+2)
	for _, field := range fields {
		if !hasParam(c, field.param) {
			continue
		}
		setClauses = append(setClauses, field.column+"=?")
		args = append(args, getParam(c, field.param))
	}
	if len(setClauses) == 0 {
		respond(c, map[string]any{"code": 400, "msg": "没有可更新的联系人字段"})
		return
	}
	setClauses = append(setClauses, "updateTime=?")
	args = append(args, time.Now().Unix(), id)
	sqlText := fmt.Sprintf("UPDATE tp_contact SET %s WHERE status=1 AND uId=?", strings.Join(setClauses, ","))
	affected, err := a.exec(sqlText, args...)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "更新联系人信息失败，请确认联系人记录存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getBannerList(c *gin.Context) {
	typeParam := getParam(c, "type")
	rows, err := a.queryRows(`SELECT id,advTitle,advImg,type FROM tp_banner WHERE type=? AND status=1`, typeParam)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该类型的轮播图数据"})
		return
	}
	baseURL := a.baseURL(c)
	arr := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		arr = append(arr, map[string]any{
			"id":     row["id"],
			"title":  row["advTitle"],
			"imgUrl": baseURL + "/static/image/" + toString(row["advImg"]),
			"type":   row["type"],
		})
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": arr, "total": len(arr)})
}

func (a *app) getNoticeList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	noticeStatus := getParam(c, "noticeStatus")
	offset := (pageNum - 1) * pageSize

	sqlText := `SELECT a.id,noticeTitle,noticeStatus,contentNotice,releaseUnit,phone,createTime,noticeName,expressId
		FROM tp_notice a JOIN tp_noticetype b ON a.expressId=b.id WHERE a.status=1`
	args := []any{}
	if noticeStatus != "" {
		sqlText += ` AND noticeStatus=?`
		args = append(args, noticeStatus)
	}
	sqlText += ` LIMIT ?,?`
	args = append(args, offset, pageSize)

	rows, err := a.queryRows(sqlText, args...)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到通知列表数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getNoticeInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT a.id,noticeTitle,noticeStatus,contentNotice,releaseUnit,phone,createTime,noticeName,expressId
		FROM tp_notice a JOIN tp_noticetype b ON a.expressId=b.id WHERE a.id=? AND a.status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该通知详情"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) readNotice(c *gin.Context) {
	id := c.Param("id")
	affected, err := a.exec(`UPDATE tp_notice SET noticeStatus=1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "通知已读更新失败，请确认通知存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getCommunityList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,name FROM tp_community WHERE status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到社区列表数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getCommunityDynamicList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,icon,title,publishTime,content FROM tp_community_dynamic
		WHERE status=1 ORDER BY publishTime DESC, id DESC LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到社区动态数据"})
		return
	}
	baseURL := a.baseURL(c)
	for i := range rows {
		rows[i]["icon"] = mediaURL(baseURL, toString(rows[i]["icon"]), "/static/image/")
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getFriendlyNeighborhoodList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,publishName,likeNum,title,publishTime,publishContent,imgUrl,userImgUrl,commentNum
		FROM tp_neighborhood WHERE status=1 ORDER BY publishTime DESC LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到友邻帖子数据"})
		return
	}
	baseURL := a.baseURL(c)
	for i := range rows {
		rows[i]["imgUrl"] = mediaURL(baseURL, localNeighborhoodPostImage(toString(rows[i]["imgUrl"])), "/static/image/")
		rows[i]["userImgUrl"] = avatarURL(baseURL, localNeighborhoodUserAvatar(toString(rows[i]["userImgUrl"])))
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) addFriendlyNeighborhoodPost(c *gin.Context) {
	title := strings.TrimSpace(getParam(c, "title"))
	content := strings.TrimSpace(getParam(c, "publishContent"))
	if content == "" {
		content = strings.TrimSpace(getParam(c, "content"))
	}
	imgURL := localNeighborhoodPostImage(strings.TrimSpace(getParam(c, "imgUrl")))
	user, err := a.getCurrentUserProfile(c)
	if err != nil || title == "" || content == "" {
		respond(c, map[string]any{"code": 400, "msg": "发帖失败，标题和内容不能为空"})
		return
	}
	if imgURL == "" {
		imgURL = "/static/image/f23f9d02-ae1e-4065-9730-42df2e539e20.jpg"
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	postID, err := a.nextNeighborhoodID()
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "生成帖子编号失败，请稍后重试"})
		return
	}
	_, err = a.exec(`INSERT INTO tp_neighborhood (id,title,publishName,publishTime,publishContent,imgUrl,userImgUrl,likeNum,commentNum,status)
		VALUES (?,?,?,?,?,?,?,?,?,1)`,
		postID, title, user["displayName"], now, content, imgURL, localNeighborhoodUserAvatar(toString(user["avatar"])), 0, 0)
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "发帖失败，请稍后重试"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": map[string]any{"id": postID}})
}

func (a *app) addFriendlyNeighborhoodComment(c *gin.Context) {
	content := getParam(c, "content")
	neighborhoodID := getParam(c, "neighborhoodId")
	user, err := a.getCurrentUserProfile(c)
	if err != nil || content == "" || neighborhoodID == "" {
		respond(c, map[string]any{"code": 400, "msg": "评论内容和 neighborhoodId 不能为空"})
		return
	}
	exists, err := a.queryOne(`SELECT id FROM tp_neighborhood WHERE id=? AND status=1`, neighborhoodID)
	if err != nil || exists == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到要评论的友邻帖子"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = a.exec(`INSERT INTO tp_neighborhood_comment (id,neighborhoodId,userName,userId,avatar,content,likeNum,publishTime,status)
		VALUES (?,?,?,?,?,?,?,?,1)`,
		newID(), neighborhoodID, user["displayName"], user["userID"], user["avatar"], content, 0, now)
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "发布评论失败，请稍后重试"})
		return
	}
	_, _ = a.exec(`UPDATE tp_neighborhood SET commentNum=commentNum+1 WHERE id=? AND status=1`, neighborhoodID)
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getFriendlyNeighborhoodInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT id,publishName,likeNum,title,publishTime,publishContent,imgUrl,userImgUrl,commentNum
		FROM tp_neighborhood WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该友邻帖子详情"})
		return
	}
	comments, err := a.queryRows(`SELECT id,userName,userId,avatar,content,likeNum,publishTime,neighborhoodId
		FROM tp_neighborhood_comment WHERE neighborhoodId=? AND status=1 ORDER BY publishTime DESC`, id)
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "查询友邻帖子评论失败"})
		return
	}

	baseURL := a.baseURL(c)
	row["imgUrl"] = mediaURL(baseURL, localNeighborhoodPostImage(toString(row["imgUrl"])), "/static/image/")
	row["userImgUrl"] = avatarURL(baseURL, localNeighborhoodUserAvatar(toString(row["userImgUrl"])))
	for i := range comments {
		comments[i]["avatar"] = avatarURL(baseURL, localNeighborhoodUserAvatar(toString(comments[i]["avatar"])))
		comments[i]["pulishTime"] = comments[i]["publishTime"]
	}
	row["userComment"] = comments
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) getCategoryList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,categoryName FROM tp_category WHERE status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到新闻分类数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getNewsList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	categoryID := getParam(c, "id")
	if categoryID == "" {
		categoryID = "1"
	}
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT a.id,categoryId,categoryName,title,subTitle,content,cover,publishDate,tags,hot,commentNum,likeNum,readNum,updateTime,createTime,remark,b.appType,top,createBy
			FROM tp_news a JOIN tp_category b ON a.categoryId=b.id
			WHERE a.categoryId=? AND a.status=1 LIMIT ?,?`, categoryID, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该分类下的新闻列表"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withNewsCover(a.baseURL(c), rows), "total": len(rows)})
}

func (a *app) getNewsAllList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT a.id,categoryId,categoryName,title,subTitle,content,cover,publishDate,tags,hot,commentNum,likeNum,readNum,updateTime,createTime,remark,b.appType,top,createBy
			FROM tp_news a JOIN tp_category b ON a.categoryId=b.id
			WHERE a.status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到新闻列表数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withNewsCover(a.baseURL(c), rows), "total": len(rows)})
}

func (a *app) getNewsInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT a.id,categoryId,categoryName,title,subTitle,content,cover,publishDate,tags,hot,commentNum,likeNum,readNum,updateTime,createTime,remark,b.appType,top,createBy
			FROM tp_news a JOIN tp_category b ON a.categoryId=b.id WHERE a.id=? AND a.status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该新闻详情"})
		return
	}
	row["cover"] = a.baseURL(c) + "/static/image/" + toString(row["cover"])
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) likeNews(c *gin.Context) {
	id := getParam(c, "id")
	if id == "" {
		id = c.Param("id")
	}
	affected, err := a.exec(`UPDATE tp_news SET likeNum=likeNum+1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 403, "msg": "新闻点赞失败，请确认新闻存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) addComment(c *gin.Context) {
	uid := mustUID(c)
	content := getParam(c, "content")
	newsID := getParam(c, "newsId")

	_, err := a.exec(`INSERT INTO tp_comment (content,newsId,userName,commentDate) VALUES (?,?,?,?)`,
		content, newsID, uid, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "发表评论失败，请确认新闻存在且参数有效"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getCommentList(c *gin.Context) {
	id := c.Param("id")
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,content,commentDate,newsId,userName,likeNum FROM tp_comment WHERE newsId=? AND status=1 LIMIT ?,?`, id, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该新闻的评论数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) likeComment(c *gin.Context) {
	id := c.Param("id")
	affected, err := a.exec(`UPDATE tp_comment SET likeNum=likeNum+1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 403, "msg": "评论点赞失败，请确认评论存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) upload(c *gin.Context) {
	fileHeader, err := getUploadFile(c, "file")
	if err != nil {
		respond(c, map[string]any{"code": 401, "msg": err.Error()})
		return
	}
	if fileHeader.Size > maxUploadSize {
		respond(c, map[string]any{"code": 401, "msg": "上传文件大小超出限制"})
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileHeader.Filename), "."))
	if ext != "jpg" && ext != "png" && ext != "jpeg" {
		respond(c, map[string]any{"code": 401, "msg": "上传文件类型不支持"})
		return
	}

	materialName := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	saveName := fileHeader.Filename
	savePath := filepath.Join("public", "storage", "uploads", saveName)
	if _, statErr := os.Stat(savePath); statErr == nil {
		saveName = fmt.Sprintf("%s_%d%s", materialName, time.Now().Unix(), filepath.Ext(fileHeader.Filename))
		savePath = filepath.Join("public", "storage", "uploads", saveName)
	}

	if err = c.SaveUploadedFile(fileHeader, savePath); err != nil {
		respond(c, map[string]any{"code": 401, "msg": err.Error()})
		return
	}

	relPath := "uploads/" + saveName
	url := "/storage/" + relPath
	respond(c, map[string]any{
		"code": 200,
		"msg":  "请求成功",
		"data": map[string]any{
			"path":         relPath,
			"avatar":       url,
			"size":         fileHeader.Size,
			"name":         saveName,
			"mime":         fileHeader.Header.Get("Content-Type"),
			"fileName":     fileHeader.Filename,
			"materialName": materialName,
		},
	})
}

func (a *app) getMaterialInfo(c *gin.Context) {
	id := c.Param("moduleId")
	row, err := a.queryOne(`SELECT id,materialName,fileName,url,createTime FROM tp_material WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该素材信息"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) getAnswerInfo(c *gin.Context) {
	a.getMaterialInfo(c)
}

func (a *app) uploadAnswerMaterial(c *gin.Context) {
	fileHeader, err := getUploadFile(c, "file")
	if err != nil {
		respond(c, map[string]any{"code": 401, "msg": err.Error()})
		return
	}
	if fileHeader.Size > maxUploadSize {
		respond(c, map[string]any{"code": 401, "msg": "上传文件大小超出限制"})
		return
	}

	materialName := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	saveName := fileHeader.Filename
	savePath := filepath.Join("public", "storage", "uploads", saveName)
	if _, statErr := os.Stat(savePath); statErr == nil {
		saveName = fmt.Sprintf("%s_%d%s", materialName, time.Now().Unix(), filepath.Ext(fileHeader.Filename))
		savePath = filepath.Join("public", "storage", "uploads", saveName)
	}
	if err = c.SaveUploadedFile(fileHeader, savePath); err != nil {
		respond(c, map[string]any{"code": 401, "msg": err.Error()})
		return
	}

	url := "/storage/uploads/" + saveName
	_, err = a.exec(`INSERT INTO tp_material (materialName,fileName,url,createTime,status) VALUES (?,?,?,?,1)`,
		materialName, saveName, url, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "上传素材记录保存失败"})
		return
	}
	respond(c, map[string]any{
		"code": 200,
		"msg":  "请求成功",
		"data": map[string]any{
			"materialName": materialName,
			"fileName":     saveName,
			"url":          url,
		},
	})
}

func (a *app) delMaterial(c *gin.Context) {
	id := c.Param("id")
	affected, err := a.exec(`UPDATE tp_material SET status=0,createTime=? WHERE id=?`, strconv.FormatInt(time.Now().Unix(), 10), id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 403, "msg": "删除素材失败，请确认素材存在"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getActivityTopList(c *gin.Context) {
	a.activityList(c, `SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop FROM tp_activity WHERE status=1 AND isTop=1 LIMIT ?,?`, 10)
}

func (a *app) getActivityList(c *gin.Context) {
	a.activityList(c, `SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop FROM tp_activity WHERE status=1 LIMIT ?,?`, 100)
}

func (a *app) getActivityCategoryList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 100)
	id := c.Param("id")
	if qid := getParam(c, "id"); qid != "" {
		id = qid
	}
	offset := (pageNum - 1) * pageSize
	rows, err := a.queryRows(`SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop
			FROM tp_activity WHERE status=1 AND category=? LIMIT ?,?`, id, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该分类下的活动数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.baseURL(c), rows), "total": len(rows)})
}

func (a *app) getActivityInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop FROM tp_activity WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该活动详情"})
		return
	}
	row["picPath"] = a.baseURL(c) + "/static/image/" + toString(row["picPath"])
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) searchActivityList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 100)
	words := getParam(c, "words")
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop
			FROM tp_activity WHERE title LIKE ? AND status=1 LIMIT ?,?`, "%"+words+"%", offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到匹配的活动数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.baseURL(c), rows), "total": len(rows)})
}

func (a *app) registerActivity(c *gin.Context) {
	activityID := getParam(c, "activityId")
	user, err := a.getCurrentUserProfile(c)
	if err != nil || activityID == "" {
		respond(c, map[string]any{"code": 400, "msg": "activityId 不能为空，请先登录后再报名"})
		return
	}
	activity, err := a.queryOne(`SELECT id,maxNum FROM tp_activity WHERE id=? AND status=1`, activityID)
	if err != nil || activity == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到要报名的活动"})
		return
	}
	existing, err := a.queryOne(`SELECT id FROM tp_activity_registration WHERE activityId=? AND userId=? AND status=1`, activityID, user["userID"])
	if err == nil && existing != nil {
		respond(c, map[string]any{"code": 200, "msg": "请求成功"})
		return
	}

	if maxNum, err := strconv.Atoi(toString(activity["maxNum"])); err == nil && maxNum > 0 {
		count, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_activity_registration WHERE activityId=? AND status=1`, activityID)
		if err == nil && count != nil {
			if total, convErr := strconv.Atoi(toString(count["total"])); convErr == nil && total >= maxNum {
				respond(c, map[string]any{"code": 400, "msg": "报名人数已满"})
				return
			}
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = a.exec(`INSERT INTO tp_activity_registration (id,activityId,userId,userName,checkedIn,evaluate,star,createTime,updateTime,status)
		VALUES (?,?,?,?,0,'',0,?,?,1)`,
		newID(), activityID, user["userID"], user["uid"], now, now)
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "活动报名失败，请稍后重试"})
		return
	}
	_ = a.syncActivitySignUpNum(activityID)
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) checkInActivity(c *gin.Context) {
	activityID := c.Param("id")
	user, err := a.getCurrentUserProfile(c)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "签到失败，请先登录"})
		return
	}
	affected, err := a.exec(`UPDATE tp_activity_registration SET checkedIn=1,updateTime=? WHERE activityId=? AND userId=? AND status=1`,
		time.Now().Format("2006-01-02 15:04:05"), activityID, user["userID"])
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "签到失败，请确认已报名该活动"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) commentActivity(c *gin.Context) {
	activityID := c.Param("id")
	evaluate := getParam(c, "evaluate")
	star := intParam(c, "star", 0)
	user, err := a.getCurrentUserProfile(c)
	if err != nil || evaluate == "" || star == 0 {
		respond(c, map[string]any{"code": 400, "msg": "活动评价内容不能为空，评分必须大于 0"})
		return
	}
	affected, err := a.exec(`UPDATE tp_activity_registration SET evaluate=?,star=?,updateTime=? WHERE activityId=? AND userId=? AND status=1`,
		evaluate, star, time.Now().Format("2006-01-02 15:04:05"), activityID, user["userID"])
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "活动评论失败，请确认已报名该活动"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) activityList(c *gin.Context, sqlText string, defaultPageSize int) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", defaultPageSize)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(sqlText, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到活动列表数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.baseURL(c), rows), "total": len(rows)})
}

func (a *app) getCourseList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,title,content,cover,video,level,duration,collection,progress FROM tp_course WHERE status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到课程列表数据"})
		return
	}
	for i := range rows {
		rows[i]["cover"] = a.baseURL(c) + "/static/image/" + toString(rows[i]["cover"])
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getCourseInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT id,title,content,cover,video,level,duration,collection,progress FROM tp_course WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到该课程详情"})
		return
	}
	chapterData, ok := a.getChapterListForCourse(mustUID(c), id)
	chapters := []map[string]any{}
	if ok {
		chapters = chapterData
	}
	row["cover"] = a.baseURL(c) + "/static/image/" + toString(row["cover"])
	row["chapter"] = chapters
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) getChapterListForCourse(uid, cid string) ([]map[string]any, bool) {
	rows, err := a.queryRows(`SELECT id,name,watch FROM tp_chapter WHERE uId=? AND cId=?`, uid, cid)
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	return rows, true
}

func (a *app) getUserByUID(uid string) (map[string]any, error) {
	return a.queryOne(`SELECT id,userName,nickName,avatar,phonenumber,sex,email,idCard,points,money,address,introduction,createTime
		FROM tp_user WHERE status=1 AND (userName=? OR phonenumber=?) LIMIT 1`, uid, uid)
}

func (a *app) getUserID(uid string) (any, error) {
	row, err := a.queryOne(`SELECT id FROM tp_user WHERE status=1 AND (userName=? OR phonenumber=?) LIMIT 1`, uid, uid)
	if err != nil || row == nil {
		return nil, fmt.Errorf("not found")
	}
	return row["id"], nil
}

func (a *app) queryRows(sqlText string, args ...any) ([]map[string]any, error) {
	rows, err := a.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (a *app) queryOne(sqlText string, args ...any) (map[string]any, error) {
	rows, err := a.queryRows(sqlText, args...)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func (a *app) exec(sqlText string, args ...any) (int64, error) {
	res, err := a.db.Exec(sqlText, args...)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return affected, nil
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		item := make(map[string]any, len(cols))
		for i, col := range cols {
			v := vals[i]
			switch vv := v.(type) {
			case []byte:
				item[col] = string(vv)
			default:
				item[col] = vv
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func getParam(c *gin.Context, key string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	if v := c.PostForm(key); v != "" {
		return v
	}
	if v := getJSONParam(c, key); v != "" {
		return v
	}
	return ""
}

func hasParam(c *gin.Context, key string) bool {
	if _, ok := c.GetQuery(key); ok {
		return true
	}
	if _, ok := c.GetPostForm(key); ok {
		return true
	}
	_, ok := getJSONBody(c)[key]
	return ok
}

func getJSONParam(c *gin.Context, key string) string {
	m := getJSONBody(c)
	if v, exists := m[key]; exists && v != nil {
		switch vv := v.(type) {
		case string:
			return vv
		case json.Number:
			return vv.String()
		default:
			return fmt.Sprintf("%v", vv)
		}
	}
	return ""
}

func getJSONBody(c *gin.Context) map[string]any {
	cached, ok := c.Get("json_body")
	if ok {
		if m, ok := cached.(map[string]any); ok {
			return m
		}
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
		empty := map[string]any{}
		c.Set("json_body", empty)
		return empty
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	m := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err = decoder.Decode(&m); err != nil {
		m = map[string]any{}
	}
	c.Set("json_body", m)
	return m
}

func intParam(c *gin.Context, key string, def int) int {
	v := getParam(c, key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getUploadFile(c *gin.Context, field string) (*multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil && err != http.ErrNotMultipart {
		return nil, err
	}
	file, header, err := c.Request.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("请上传文件")
	}
	_ = file.Close()
	return header, nil
}

func mustUID(c *gin.Context) string {
	v, _ := c.Get("uid")
	return fmt.Sprintf("%v", v)
}

func respond(c *gin.Context, payload map[string]any) {
	c.JSON(http.StatusOK, payload)
}

func withNewsCover(httpURL string, rows []map[string]any) []map[string]any {
	for i := range rows {
		rows[i]["cover"] = httpURL + "/static/image/" + toString(rows[i]["cover"])
	}
	return rows
}

func withActivityPic(httpURL string, rows []map[string]any) []map[string]any {
	for i := range rows {
		rows[i]["picPath"] = httpURL + "/static/image/" + toString(rows[i]["picPath"])
	}
	return rows
}

func avatarURL(baseURL, avatar string) string {
	avatar = strings.TrimSpace(avatar)
	switch {
	case avatar == "":
		return ""
	case strings.HasPrefix(avatar, "http://"), strings.HasPrefix(avatar, "https://"):
		return avatar
	case strings.HasPrefix(avatar, "/"):
		return baseURL + avatar
	default:
		return baseURL + "/static/avatar/" + avatar
	}
}

func localNeighborhoodPostImage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/static/image/") {
		return raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	name := pathBase(raw)
	switch name {
	case "news_hot.png", "Gk08RijaAAAyuFq.png":
		return "/static/image/f23f9d02-ae1e-4065-9730-42df2e539e20.jpg"
	default:
		if name != "" {
			return "/static/image/" + name
		}
		return "/static/image/f23f9d02-ae1e-4065-9730-42df2e539e20.jpg"
	}
}

func localNeighborhoodUserAvatar(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "/static/avatar/") || strings.HasPrefix(raw, "/static/image/") {
		return raw
	}
	return "/static/avatar/avatar1.png"
}

func mediaURL(baseURL, raw, defaultPrefix string) string {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return ""
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		return raw
	case strings.HasPrefix(raw, "/"):
		return baseURL + raw
	default:
		return baseURL + defaultPrefix + raw
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func (a *app) baseURL(c *gin.Context) string {
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host, port := splitHostPort(c.Request.Host)
	if host == "" || port == "" {
		cfgHost, cfgPort := hostPortFromURL(a.httpURL)
		if host == "" {
			host = cfgHost
		}
		if port == "" {
			port = cfgPort
		}
	}

	if ip := localIPv4(); ip != "" {
		host = ip
	}
	if host == "" {
		return a.httpURL
	}
	if port != "" {
		return scheme + "://" + net.JoinHostPort(host, port)
	}
	return scheme + "://" + host
}

func hostPortFromURL(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	return splitHostPort(u.Host)
}

func splitHostPort(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", ""
		}
		raw = u.Host
	}
	host, port, err := net.SplitHostPort(raw)
	if err == nil {
		return host, port
	}
	if strings.Count(raw, ":") == 0 {
		return raw, ""
	}
	return strings.Trim(raw, "[]"), ""
}

func localIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func generateSMSCode() (string, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := int(b[0])<<8 | int(b[1])
	return fmt.Sprintf("%04d", 1000+n%9000), nil
}

func newID() int64 {
	return time.Now().UnixNano()
}

func (a *app) getCurrentUserProfile(c *gin.Context) (map[string]any, error) {
	uid := mustUID(c)
	row, err := a.getUserByUID(uid)
	if err != nil || row == nil {
		return nil, fmt.Errorf("not found")
	}
	displayName := toString(row["nickName"])
	if displayName == "" {
		displayName = toString(row["userName"])
	}
	return map[string]any{
		"uid":         uid,
		"userID":      row["id"],
		"userName":    row["userName"],
		"displayName": displayName,
		"avatar":      row["avatar"],
	}, nil
}

func (a *app) ensureSupplementalTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tp_neighborhood (
			id BIGINT PRIMARY KEY,
			title TEXT,
			publishName TEXT,
			publishTime TEXT,
			publishContent TEXT,
			imgUrl TEXT,
			userImgUrl TEXT,
			likeNum INTEGER DEFAULT 0,
			commentNum INTEGER DEFAULT 0,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_neighborhood_comment (
			id BIGINT PRIMARY KEY,
			neighborhoodId BIGINT,
			userName TEXT,
			userId BIGINT,
			avatar TEXT,
			content TEXT,
			likeNum INTEGER DEFAULT 0,
			publishTime TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_activity_registration (
			id BIGINT PRIMARY KEY,
			activityId BIGINT,
			userId BIGINT,
			userName TEXT,
			checkedIn INTEGER DEFAULT 0,
			evaluate TEXT,
			star INTEGER DEFAULT 0,
			createTime TEXT,
			updateTime TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_community_dynamic (
			id INTEGER PRIMARY KEY,
			icon TEXT,
			title TEXT,
			publishTime TEXT,
			content TEXT,
			status TEXT
		)`,
	}
	for _, stmt := range stmts {
		if _, err := a.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := a.seedNeighborhoodData(); err != nil {
		return err
	}
	if err := a.seedCommunityDynamicData(); err != nil {
		return err
	}
	if err := a.normalizeNeighborhoodIDs(); err != nil {
		return err
	}
	return a.normalizeNeighborhoodMedia()
}

func (a *app) seedNeighborhoodData() error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_neighborhood WHERE status=1`)
	if err == nil && row != nil && toString(row["total"]) != "0" {
		return nil
	}

	users, err := a.queryRows(`SELECT id,userName,nickName,avatar FROM tp_user WHERE status=1 ORDER BY id LIMIT 2`)
	if err != nil {
		return err
	}
	type seedItem struct {
		Title   string
		Content string
		Image   string
	}
	items := []seedItem{
		{Title: "今天天气真好啊", Content: "夏天的风我永远记得，清清楚楚地说要热死我。欢迎大家出来散步、晒太阳、拍照打卡。", Image: "/storage/uploads/news_hot.png"},
		{Title: "社区便民活动预告", Content: "周末广场将举行便民服务和公益宣传活动，欢迎大家带上家人一起参加。", Image: "/storage/uploads/Gk08RijaAAAyuFq.png"},
	}
	now := time.Now()
	for i, item := range items {
		publishName := "社区居民"
		userImg := fmt.Sprintf("/static/avatar/avatar%d.png", i+1)
		if i < len(users) {
			publishName = toString(users[i]["nickName"])
			if publishName == "" {
				publishName = toString(users[i]["userName"])
			}
		}
		if _, err = a.exec(`INSERT INTO tp_neighborhood (id,title,publishName,publishTime,publishContent,imgUrl,userImgUrl,likeNum,commentNum,status)
			VALUES (?,?,?,?,?,?,?,?,?,1)`,
			i+1, item.Title, publishName, now.Add(-time.Duration(i)*time.Hour).Format("2006-01-02 15:04:05"),
			item.Content, localNeighborhoodPostImage(item.Image), userImg, 0, 0); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) normalizeNeighborhoodMedia() error {
	rows, err := a.queryRows(`SELECT id,imgUrl,userImgUrl FROM tp_neighborhood`)
	if err != nil {
		return err
	}
	for idx, row := range rows {
		userAvatar := localNeighborhoodUserAvatar(toString(row["userImgUrl"]))
		if userAvatar == "/static/avatar/avatar1.png" {
			userAvatar = fmt.Sprintf("/static/avatar/avatar%d.png", idx%13+1)
		}
		if _, err = a.exec(`UPDATE tp_neighborhood SET imgUrl=?,userImgUrl=? WHERE id=?`,
			localNeighborhoodPostImage(toString(row["imgUrl"])), userAvatar, row["id"]); err != nil {
			return err
		}
	}
	if _, err = a.exec(`UPDATE tp_neighborhood_comment SET avatar='/static/avatar/avatar1.png' WHERE avatar NOT LIKE '/static/%' OR avatar IS NULL OR avatar=''`); err != nil {
		return err
	}
	return nil
}

func (a *app) normalizeNeighborhoodIDs() error {
	rows, err := a.queryRows(`SELECT id FROM tp_neighborhood ORDER BY publishTime ASC, id ASC`)
	if err != nil {
		return err
	}
	type pair struct {
		oldID string
		newID int
	}
	pairs := make([]pair, 0, len(rows))
	for i, row := range rows {
		oldID := toString(row["id"])
		if oldID == strconv.Itoa(i) {
			continue
		}
		pairs = append(pairs, pair{oldID: oldID, newID: i})
	}
	if len(pairs) == 0 {
		return nil
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range pairs {
		tmpID := fmt.Sprintf("tmp_%d", item.newID)
		if _, err = tx.Exec(`UPDATE tp_neighborhood SET id=? WHERE id=?`, tmpID, item.oldID); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE tp_neighborhood_comment SET neighborhoodId=? WHERE neighborhoodId=?`, tmpID, item.oldID); err != nil {
			return err
		}
	}
	for _, item := range pairs {
		tmpID := fmt.Sprintf("tmp_%d", item.newID)
		if _, err = tx.Exec(`UPDATE tp_neighborhood SET id=? WHERE id=?`, item.newID, tmpID); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE tp_neighborhood_comment SET neighborhoodId=? WHERE neighborhoodId=?`, item.newID, tmpID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *app) nextNeighborhoodID() (int, error) {
	row, err := a.queryOne(`SELECT COALESCE(MAX(CAST(id AS INTEGER)), -1) AS maxID FROM tp_neighborhood`)
	if err != nil || row == nil {
		return 0, err
	}
	maxID, convErr := strconv.Atoi(toString(row["maxID"]))
	if convErr != nil {
		return 0, convErr
	}
	return maxID + 1, nil
}

func ensureStaticAssets() error {
	if err := copyDirIfMissing(filepath.Join("cmd", "static", "avatar"), filepath.Join("public", "static", "avatar")); err != nil {
		return err
	}
	return copyDirIfMissing(filepath.Join("cmd", "static", "image"), filepath.Join("public", "static", "image"))
}

func copyDirIfMissing(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func pathBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		raw = u.Path
	}
	return filepath.Base(raw)
}

func (a *app) syncActivitySignUpNum(activityID string) error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_activity_registration WHERE activityId=? AND status=1`, activityID)
	if err != nil || row == nil {
		return err
	}
	_, err = a.exec(`UPDATE tp_activity SET signUpNum=? WHERE id=? AND status=1`, toString(row["total"]), activityID)
	return err
}

func (a *app) seedCommunityDynamicData() error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_community_dynamic WHERE status=1`)
	if err == nil && row != nil && toString(row["total"]) != "0" {
		return nil
	}
	type item struct {
		Icon    string
		Title   string
		Content string
	}
	items := []item{
		{
			Icon:    "client-img-1.jpg",
			Title:   "社区晨练活动恢复开放",
			Content: "本周起，社区广场晨练活动恢复开放，时间为每天早上 6:30 到 8:00，请居民有序参与并注意安全。",
		},
		{
			Icon:    "client-img-2.jpg",
			Title:   "周末义诊服务进小区",
			Content: "社区联合卫生服务站将在周六上午开展义诊服务，提供血压测量、健康咨询和用药指导。",
		},
		{
			Icon:    "client-img-3.jpg",
			Title:   "垃圾分类宣传周启动",
			Content: "社区本周启动垃圾分类宣传周，将开展入户讲解、知识答题和兑换活动，欢迎居民积极参与。",
		},
	}
	now := time.Now()
	for i, item := range items {
		if _, err = a.exec(`INSERT INTO tp_community_dynamic (id,icon,title,publishTime,content,status)
			VALUES (?,?,?,?,?,1)`,
			i, item.Icon, item.Title, now.Add(-time.Duration(i)*2*time.Hour).Format("2006-01-02 15:04:05"), item.Content); err != nil {
			return err
		}
	}
	return nil
}
