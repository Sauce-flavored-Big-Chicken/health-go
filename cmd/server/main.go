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
	defaultImage   = "/static/image/f23f9d02-ae1e-4065-9730-42df2e539e20.jpg"
	defaultAvatar  = "/static/avatar/avatar1.png"
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
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		respond(c, map[string]any{"code": 500, "msg": "服务内部错误，请稍后重试"})
		c.Abort()
	}))
	r.Use(corsMiddleware())
	r.HandleMethodNotAllowed = true

	r.NoRoute(func(c *gin.Context) {
		respond(c, map[string]any{"code": 404, "msg": "接口不存在，请检查请求路径"})
	})
	r.NoMethod(func(c *gin.Context) {
		respond(c, map[string]any{"code": 405, "msg": "请求方法不支持，请检查请求方式"})
	})

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
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/static/docs/index.html")
	})
	r.GET("/docs/api.md", func(c *gin.Context) {
		c.File("智慧健康API接口文档V1.0(3).md")
	})

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
	auth.GET("/prod-api/api/common/datacard", a.getDataCardList)
	auth.GET("/prod-api/api/question/questionList/:id/:level", a.getQuestionList)
	auth.POST("/prod-api/api/question/submit", a.submitQuestionAnswer)
	auth.GET("/prod-api/api/question/statistics", a.getQuestionStatistics)
	auth.GET("/prod-api/api/data/list_1", a.getDataList1)
	auth.GET("/prod-api/api/data/list_2", a.getDataList2)
	auth.GET("/prod-api/api/data/list_3", a.getDataList3)
	auth.GET("/prod-api/api/data/list_4", a.getDataList4)

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
	if !isValidPhoneNumber(getParam(c, "phonenumber")) {
		respond(c, map[string]any{"code": 400, "msg": "手机号格式错误，必须为11位数字"})
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
	row["avatar"] = userAvatarURL(a.baseURL(c), toString(row["avatar"]))
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
		if field.param == "phonenumber" && !isValidPhoneNumber(getParam(c, "phonenumber")) {
			respond(c, map[string]any{"code": 400, "msg": "手机号格式错误，必须为11位数字"})
			return
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
	baseURL := a.baseURL(c)
	avatar := []map[string]any{
		{"id": 1, "avatar": "avatar1.png", "avatarUrl": baseURL + "/static/avatar/avatar1.png"},
		{"id": 2, "avatar": "avatar2.png", "avatarUrl": baseURL + "/static/avatar/avatar2.png"},
		{"id": 3, "avatar": "avatar3.png", "avatarUrl": baseURL + "/static/avatar/avatar3.png"},
		{"id": 4, "avatar": "avatar4.png", "avatarUrl": baseURL + "/static/avatar/avatar4.png"},
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
	row, err := a.queryOne(`SELECT name,relationship,telephone,alternatePhone,createTime FROM tp_contact WHERE status=1 AND uId=?`, id)
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
		{param: "name", column: "name"},
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

func (a *app) getDataCardList(c *gin.Context) {
	rows, err := a.queryRows(`SELECT id,title,num,unit,icon,trend FROM tp_datacard WHERE status=1 ORDER BY id`)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到数据卡片数据"})
		return
	}
	baseURL := a.baseURL(c)
	for i := range rows {
		rows[i]["icon"] = mediaURL(baseURL, toString(rows[i]["icon"]), "/static/image/")
		rows[i]["trend"] = mediaURL(baseURL, toString(rows[i]["trend"]), "/static/image/")
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getQuestionList(c *gin.Context) {
	moduleID := c.Param("id")
	level := c.Param("level")
	limit := intParam(c, "count", 5)
	if limit <= 0 {
		limit = 5
	}
	rows, err := a.queryRows(`SELECT id,questionType,question,optionA,optionB,optionC,optionD,optionE,optionF,answer,analysis,parseText,score
		FROM tp_question_bank WHERE status=1 AND moduleId=? AND level=? ORDER BY RANDOM() LIMIT ?`, moduleID, level, limit)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该题库等级的题目"})
		return
	}
	for i := range rows {
		parseText := strings.TrimSpace(toString(rows[i]["parseText"]))
		analysis := strings.TrimSpace(toString(rows[i]["analysis"]))
		switch {
		case parseText == "" && analysis != "":
			rows[i]["parseText"] = analysis
		}
		delete(rows[i], "analysis")
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) submitQuestionAnswer(c *gin.Context) {
	uid := mustUID(c)
	if uid == "" {
		respond(c, map[string]any{"code": 400, "msg": "提交答案失败，请先登录"})
		return
	}
	qID := getParam(c, "qId")
	answer := getParam(c, "answer")
	scoreStr := strings.TrimSpace(getParam(c, "score"))
	if qID == "" || answer == "" {
		respond(c, map[string]any{"code": 400, "msg": "qId、answer 不能为空"})
		return
	}
	score := 0
	var err error
	if scoreStr != "" {
		score, err = strconv.Atoi(scoreStr)
		if err != nil {
			respond(c, map[string]any{"code": 400, "msg": "score 必须是数字"})
			return
		}
	}
	question, err := a.queryOne(`SELECT id,moduleId,level,answer,score FROM tp_question_bank WHERE id=? AND status=1`, qID)
	if err != nil || question == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到要提交的题目"})
		return
	}
	correctAnswer := strings.TrimSpace(toString(question["answer"]))
	questionScore := intFromAny(question["score"])
	isCorrect := strings.EqualFold(strings.TrimSpace(answer), correctAnswer)
	if isCorrect && score == 0 {
		score = questionScore
	}
	if !isCorrect && score != 0 {
		score = 0
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = a.exec(`INSERT INTO tp_question_record (id,userUid,qId,moduleId,level,answer,isCorrect,score,createTime,status)
		VALUES (?,?,?,?,?,?,?,?,?,1)`,
		newID(), uid, qID, toString(question["moduleId"]), toString(question["level"]), answer, boolToInt(isCorrect), score, now)
	if err != nil {
		respond(c, map[string]any{"code": 403, "msg": "提交答案失败，请稍后重试"})
		return
	}
	respond(c, map[string]any{
		"code": 200,
		"msg":  "请求成功",
		"data": map[string]any{
			"qId":           qID,
			"userAnswer":    answer,
			"isCorrect":     boolToInt(isCorrect),
			"correctAnswer": correctAnswer,
			"score":         score,
		},
	})
}

func (a *app) getQuestionStatistics(c *gin.Context) {
	uid := mustUID(c)
	if uid == "" {
		respond(c, map[string]any{"code": 400, "msg": "获取答题统计失败，请先登录"})
		return
	}
	moduleID := strings.TrimSpace(getParam(c, "moduleId"))
	if moduleID == "" {
		moduleID = "1"
	}

	totalAnswered := 0
	totalCorrect := 0
	totalWrong := 0
	todayAnswered := 0
	today := time.Now().Format("2006-01-02")

	totalRow, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=?`, uid, moduleID)
	if err == nil && totalRow != nil {
		totalAnswered = intFromAny(totalRow["total"])
	}
	correctRow, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=? AND isCorrect=1`, uid, moduleID)
	if err == nil && correctRow != nil {
		totalCorrect = intFromAny(correctRow["total"])
	}
	totalWrong = totalAnswered - totalCorrect
	if totalWrong < 0 {
		totalWrong = 0
	}
	todayRow, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=? AND substr(createTime,1,10)=?`, uid, moduleID, today)
	if err == nil && todayRow != nil {
		todayAnswered = intFromAny(todayRow["total"])
	}

	accuracy := 0.0
	if totalAnswered > 0 {
		accuracy = float64(totalCorrect) * 100.0 / float64(totalAnswered)
	}

	levelRows, err := a.queryRows(`SELECT DISTINCT level FROM tp_question_bank WHERE status=1 AND moduleId=? ORDER BY level`, moduleID)
	if err != nil || len(levelRows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到该模块题库，无法统计"})
		return
	}

	progress := make([]map[string]any, 0, len(levelRows))
	for _, lv := range levelRows {
		level := strings.TrimSpace(toString(lv["level"]))
		if level == "" {
			continue
		}

		totalQuestions := 0
		completedQuestions := 0
		levelCorrect := 0
		levelWrong := 0

		levelTotalRow, qErr := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_bank WHERE status=1 AND moduleId=? AND level=?`, moduleID, level)
		if qErr == nil && levelTotalRow != nil {
			totalQuestions = intFromAny(levelTotalRow["total"])
		}
		completedRow, qErr := a.queryOne(`SELECT COUNT(DISTINCT qId) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=? AND level=?`, uid, moduleID, level)
		if qErr == nil && completedRow != nil {
			completedQuestions = intFromAny(completedRow["total"])
		}
		correctLevelRow, qErr := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=? AND level=? AND isCorrect=1`, uid, moduleID, level)
		if qErr == nil && correctLevelRow != nil {
			levelCorrect = intFromAny(correctLevelRow["total"])
		}
		wrongLevelRow, qErr := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_record WHERE status=1 AND userUid=? AND moduleId=? AND level=? AND isCorrect=0`, uid, moduleID, level)
		if qErr == nil && wrongLevelRow != nil {
			levelWrong = intFromAny(wrongLevelRow["total"])
		}

		progressPercent := 0.0
		if totalQuestions > 0 {
			progressPercent = float64(completedQuestions) * 100.0 / float64(totalQuestions)
		}

		progress = append(progress, map[string]any{
			"level":              level,
			"totalQuestions":     totalQuestions,
			"completedQuestions": completedQuestions,
			"correctCount":       levelCorrect,
			"wrongCount":         levelWrong,
			"progressPercent":    fmt.Sprintf("%.2f%%", progressPercent),
		})
	}

	respond(c, map[string]any{
		"code": 200,
		"msg":  "请求成功",
		"data": map[string]any{
			"answerAccuracyPercent": fmt.Sprintf("%.2f%%", accuracy),
			"totalAnswered":         totalAnswered,
			"totalWrong":            totalWrong,
			"todayAnswered":         todayAnswered,
			"levelProgress":         progress,
		},
	})
}

func (a *app) getDataList1(c *gin.Context) {
	series, err := a.pollutionSeries([]string{"aqi", "pm2_5", "pm10", "so2", "no2", "co"})
	if err != nil || len(series) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到污染物趋势数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": series})
}

func (a *app) getDataList2(c *gin.Context) {
	series, err := a.pollutionSeries([]string{"pm2_5", "pm10", "so2", "no2"})
	if err != nil || len(series) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到污染物对比数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": series})
}

func (a *app) getDataList3(c *gin.Context) {
	series, err := a.pollutionSeries([]string{"aqi", "pm2_5", "pm10", "so2", "no2", "co"})
	if err != nil || len(series) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "未找到污染物分析数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": series})
}

func (a *app) getDataList4(c *gin.Context) {
	row, err := a.queryOne(`SELECT aqi,pm2_5,pm10,so2,no2,co FROM tp_pollution_daily WHERE status=1 ORDER BY recordDate DESC LIMIT 1`)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "未找到污染物最新数据"})
		return
	}
	data := []map[string]any{
		{"name": "aqi", "data": numberFromAny(row["aqi"])},
		{"name": "pm2.5", "data": numberFromAny(row["pm2_5"])},
		{"name": "pm10", "data": numberFromAny(row["pm10"])},
		{"name": "so2", "data": numberFromAny(row["so2"])},
		{"name": "no2", "data": numberFromAny(row["no2"])},
		{"name": "co", "data": numberFromAny(row["co"])},
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": data})
}

func (a *app) pollutionSeries(fields []string) ([]map[string]any, error) {
	rows, err := a.queryRows(`SELECT recordDate,aqi,pm2_5,pm10,so2,no2,co
		FROM tp_pollution_daily WHERE status=1 ORDER BY recordDate DESC LIMIT 7`)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	// API expects chronological order.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		values := make([]any, 0, len(rows))
		for _, row := range rows {
			values = append(values, numberFromAny(row[field]))
		}
		out = append(out, map[string]any{
			"name": displayPollutionField(field),
			"data": values,
		})
	}
	return out, nil
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
	row["cover"] = newsCoverURL(a.baseURL(c), toString(row["cover"]))
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
	fullURL := mediaURL(a.baseURL(c), url, "")
	respond(c, map[string]any{
		"code": 200,
		"msg":  "请求成功",
		"data": map[string]any{
			"path":         relPath,
			"avatar":       fullURL,
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
	row["url"] = mediaURL(a.baseURL(c), toString(row["url"]), "")
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
			"url":          mediaURL(a.baseURL(c), url, ""),
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
		rows[i]["cover"] = newsCoverURL(httpURL, toString(rows[i]["cover"]))
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

func userAvatarURL(baseURL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return baseURL + defaultAvatar
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if localPath, ok := localMediaPathFromAbsoluteURL(baseURL, raw); ok && !mediaFileExists(localPath) {
			return baseURL + defaultAvatar
		}
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		if isLocalMediaPath(raw) && !mediaFileExists(raw) {
			return baseURL + defaultAvatar
		}
		return baseURL + raw
	}
	candidate := "/static/avatar/" + raw
	if !mediaFileExists(candidate) {
		return baseURL + defaultAvatar
	}
	return baseURL + candidate
}

func newsCoverURL(baseURL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return baseURL + defaultImage
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if localPath, ok := localMediaPathFromAbsoluteURL(baseURL, raw); ok && !mediaFileExists(localPath) {
			return baseURL + defaultImage
		}
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		if isLocalMediaPath(raw) && !mediaFileExists(raw) {
			return baseURL + defaultImage
		}
		return baseURL + raw
	}
	candidate := "/static/image/" + raw
	if !mediaFileExists(candidate) {
		return baseURL + defaultImage
	}
	return baseURL + candidate
}

func isLocalMediaPath(p string) bool {
	return strings.HasPrefix(p, "/static/") || strings.HasPrefix(p, "/storage/")
}

func localMediaPathFromAbsoluteURL(baseURL, raw string) (string, bool) {
	mediaURL, err := url.Parse(raw)
	if err != nil || mediaURL.Host == "" || !isLocalMediaPath(mediaURL.Path) {
		return "", false
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return "", false
	}
	if !strings.EqualFold(mediaURL.Hostname(), base.Hostname()) {
		return "", false
	}
	if effectivePort(mediaURL) != effectivePort(base) {
		return "", false
	}
	return mediaURL.Path, true
}

func effectivePort(u *url.URL) string {
	port := u.Port()
	if port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	default:
		return "80"
	}
}

func mediaFileExists(p string) bool {
	if !isLocalMediaPath(p) {
		return true
	}
	rel := strings.TrimPrefix(p, "/")
	fsPath := filepath.Join("public", filepath.FromSlash(rel))
	_, err := os.Stat(fsPath)
	return err == nil
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
		return defaultImage
	default:
		if name != "" {
			return "/static/image/" + name
		}
		return defaultImage
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

func intFromAny(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case float64:
		return int(vv)
	case []byte:
		n, _ := strconv.Atoi(string(vv))
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprintf("%v", vv))
		return n
	}
}

func numberFromAny(v any) any {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return vv
	case float64:
		// Keep integers clean in JSON output.
		if vv == float64(int64(vv)) {
			return int64(vv)
		}
		return vv
	case []byte:
		s := string(vv)
		if strings.Contains(s, ".") {
			f, err := strconv.ParseFloat(s, 64)
			if err == nil {
				if f == float64(int64(f)) {
					return int64(f)
				}
				return f
			}
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		return s
	default:
		s := fmt.Sprintf("%v", vv)
		if strings.Contains(s, ".") {
			f, err := strconv.ParseFloat(s, 64)
			if err == nil {
				if f == float64(int64(f)) {
					return int64(f)
				}
				return f
			}
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		return s
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func displayPollutionField(name string) string {
	switch name {
	case "pm2_5":
		return "pm2.5"
	default:
		return name
	}
}

func isValidPhoneNumber(phone string) bool {
	phone = strings.TrimSpace(phone)
	if len(phone) != 11 {
		return false
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
		`CREATE TABLE IF NOT EXISTS tp_datacard (
			id INTEGER PRIMARY KEY,
			title TEXT,
			num TEXT,
			unit TEXT,
			icon TEXT,
			trend TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_question_bank (
			id INTEGER PRIMARY KEY,
			moduleId TEXT,
			level TEXT,
			questionType TEXT,
			question TEXT,
			optionA TEXT,
			optionB TEXT,
			optionC TEXT,
			optionD TEXT,
			optionE TEXT,
			optionF TEXT,
			answer TEXT,
			analysis TEXT,
			parseText TEXT,
			score INTEGER,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_question_record (
			id BIGINT PRIMARY KEY,
			userUid TEXT,
			qId INTEGER,
			moduleId TEXT,
			level TEXT,
			answer TEXT,
			isCorrect INTEGER DEFAULT 0,
			score INTEGER DEFAULT 0,
			createTime TEXT,
			status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tp_pollution_daily (
			id INTEGER PRIMARY KEY,
			recordDate TEXT,
			aqi REAL,
			pm2_5 REAL,
			pm10 REAL,
			so2 REAL,
			no2 REAL,
			co REAL,
			status TEXT
		)`,
	}
	for _, stmt := range stmts {
		if _, err := a.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := a.ensureContactNameColumn(); err != nil {
		return err
	}
	if err := a.ensureQuestionParseTextColumn(); err != nil {
		return err
	}
	if err := a.seedNeighborhoodData(); err != nil {
		return err
	}
	if err := a.seedCommunityDynamicData(); err != nil {
		return err
	}
	if err := a.seedDataCardData(); err != nil {
		return err
	}
	if err := a.seedQuestionBankData(); err != nil {
		return err
	}
	if err := a.seedPollutionData(); err != nil {
		return err
	}
	if err := a.normalizeNeighborhoodIDs(); err != nil {
		return err
	}
	return a.normalizeNeighborhoodMedia()
}

func (a *app) ensureContactNameColumn() error {
	_, err := a.db.Exec(`ALTER TABLE tp_contact ADD COLUMN name TEXT`)
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists") {
		return nil
	}
	return err
}

func (a *app) ensureQuestionParseTextColumn() error {
	_, err := a.db.Exec(`ALTER TABLE tp_question_bank ADD COLUMN parseText TEXT`)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if !strings.Contains(msg, "duplicate column") && !strings.Contains(msg, "already exists") {
			return err
		}
	}
	if _, err = a.db.Exec(`UPDATE tp_question_bank SET parseText=analysis WHERE (parseText IS NULL OR TRIM(parseText)='') AND analysis IS NOT NULL AND TRIM(analysis)<>''`); err != nil {
		return err
	}
	if _, err = a.db.Exec(`UPDATE tp_question_bank SET analysis=parseText WHERE (analysis IS NULL OR TRIM(analysis)='') AND parseText IS NOT NULL AND TRIM(parseText)<>''`); err != nil {
		return err
	}
	return nil
}

func (a *app) seedDataCardData() error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_datacard WHERE status=1`)
	if err == nil && row != nil && toString(row["total"]) != "0" {
		return nil
	}
	items := []struct {
		ID    int
		Title string
		Num   string
		Unit  string
	}{
		{ID: 1, Title: "AQI 指数", Num: "45", Unit: "优"},
		{ID: 2, Title: "PM2.5", Num: "22", Unit: "μg/m³"},
		{ID: 3, Title: "PM10", Num: "48", Unit: "μg/m³"},
		{ID: 4, Title: "SO2", Num: "10", Unit: "μg/m³"},
	}
	for _, item := range items {
		if _, err = a.exec(`INSERT INTO tp_datacard (id,title,num,unit,icon,trend,status) VALUES (?,?,?,?,?,?,1)`,
			item.ID, item.Title, item.Num, item.Unit, defaultImage, defaultImage); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) seedQuestionBankData() error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_question_bank WHERE status=1`)
	if err == nil && row != nil && toString(row["total"]) != "0" {
		return nil
	}
	type q struct {
		ID       int
		ModuleID string
		Level    string
		Type     string
		Question string
		A, B, C  string
		D, E, F  string
		Answer   string
		Analysis string
		Parse    string
		Score    int
	}
	items := []q{
		{1, "1", "1", "4", "PHP 的数组长度可通过 count() 获取。", "正确", "错误", "", "", "", "", "A", "count() 可以统计数组元素数量。", "count() 可以统计数组元素数量。", 2},
		{2, "1", "1", "4", "Go 语言中 map 是线程安全的。", "正确", "错误", "", "", "", "", "B", "原生 map 非线程安全，需要加锁或并发安全容器。", "原生 map 非线程安全，需要加锁或并发安全容器。", 2},
		{3, "1", "1", "4", "SQL 的 WHERE 条件可以省略。", "正确", "错误", "", "", "", "", "A", "无 WHERE 时会影响整表。", "无 WHERE 时会影响整表。", 1},
		{4, "1", "1", "1", "HTTP 常见成功状态码是？", "200", "404", "500", "302", "", "", "A", "200 表示请求成功。", "200 表示请求成功。", 2},
		{5, "1", "1", "1", "以下哪个是关系型数据库？", "Redis", "MongoDB", "MySQL", "Kafka", "", "", "C", "MySQL 是关系型数据库。", "MySQL 是关系型数据库。", 2},
		{6, "1", "2", "1", "JWT 常用于？", "静态资源压缩", "用户认证授权", "数据库分片", "图像处理", "", "", "B", "JWT 常用于认证和鉴权。", "JWT 常用于认证和鉴权。", 2},
		{7, "1", "2", "4", "RESTful API 通常使用不同 HTTP 方法表达语义。", "正确", "错误", "", "", "", "", "A", "GET/POST/PUT/DELETE 语义明确。", "GET/POST/PUT/DELETE 语义明确。", 2},
		{8, "1", "2", "1", "下列哪项最适合作为密码存储方式？", "明文", "MD5 无盐", "带盐哈希", "Base64", "", "", "C", "应使用带盐哈希。", "应使用带盐哈希。", 3},
	}
	for _, item := range items {
		if _, err = a.exec(`INSERT INTO tp_question_bank
			(id,moduleId,level,questionType,question,optionA,optionB,optionC,optionD,optionE,optionF,answer,analysis,parseText,score,status)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1)`,
			item.ID, item.ModuleID, item.Level, item.Type, item.Question,
			item.A, item.B, item.C, item.D, item.E, item.F, item.Answer, item.Analysis, item.Parse, item.Score); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) seedPollutionData() error {
	row, err := a.queryOne(`SELECT COUNT(1) AS total FROM tp_pollution_daily WHERE status=1`)
	if err == nil && row != nil && toString(row["total"]) != "0" {
		return nil
	}
	type daily struct {
		AQI, PM25, PM10, SO2, NO2, CO float64
	}
	values := []daily{
		{65, 22, 58, 9, 31, 1.1},
		{82, 31, 75, 11, 36, 1.2},
		{95, 38, 88, 12, 40, 1.3},
		{78, 27, 70, 10, 34, 1.1},
		{105, 45, 92, 10, 45, 1.5},
		{88, 33, 81, 9, 39, 1.2},
		{70, 24, 64, 8, 30, 1.0},
	}
	start := time.Now().AddDate(0, 0, -(len(values) - 1))
	for i, d := range values {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		if _, err = a.exec(`INSERT INTO tp_pollution_daily
			(id,recordDate,aqi,pm2_5,pm10,so2,no2,co,status)
			VALUES (?,?,?,?,?,?,?,?,1)`,
			i+1, day, d.AQI, d.PM25, d.PM10, d.SO2, d.NO2, d.CO); err != nil {
			return err
		}
	}
	return nil
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
