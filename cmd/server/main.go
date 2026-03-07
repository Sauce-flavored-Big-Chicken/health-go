package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

	auth := r.Group("/")
	auth.Use(a.authMiddleware())

	auth.GET("/prod-api/api/user/getUserInfo", a.getUserInfo)
	auth.PUT("/prod-api/api/user/updateUserInfo", a.updateUserInfo)
	auth.PUT("/prod-api/api/user/resetPwd", a.resetPwd)
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

	auth.GET("/prod-api/api/press/category/list", a.getCategoryList)
	auth.GET("/prod-api/api/press/category/newsList", a.getNewsList)
	auth.GET("/prod-api/api/press/newsList", a.getNewsAllList)
	auth.GET("/prod-api/api/press/news/:id", a.getNewsInfo)
	auth.PUT("/prod-api/api/press/like", a.likeNews)

	auth.POST("/prod-api/api/comment/pressComment", a.addComment)
	auth.GET("/prod-api/api/comment/comment/:id", a.getCommentList)
	auth.PUT("/prod-api/api/comment/like/:id", a.likeComment)

	auth.POST("/prod-api/api/common/upload", a.upload)

	auth.GET("/prod-api/api/activity/topList", a.getActivityTopList)
	auth.GET("/prod-api/api/activity/list", a.getActivityList)
	auth.GET("/prod-api/api/activity/category/list/:id", a.getActivityCategoryList)
	auth.GET("/prod-api/api/activity/:id", a.getActivityInfo)
	auth.POST("/prod-api/api/activity/search", a.searchActivityList)

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
			respond(c, map[string]any{"code": 401, "msg": "request must with token"})
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
			res.Msg = "签名不正确"
		case strings.Contains(msg, "expired"):
			res.Msg = "token失效"
		case strings.Contains(msg, "not valid yet"):
			res.Msg = "token失效"
		default:
			res.Msg = "未知错误"
		}
		return res
	}
	if !parsed.Valid {
		res.Msg = "未知错误"
		return res
	}
	data, ok := claims["data"].(map[string]any)
	if !ok {
		res.Msg = "未知错误"
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

	row, err := a.queryOne(`SELECT id FROM tp_user WHERE userName=? AND passWord=? AND status=1`, userName, passWord)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "token": a.generateToken(userName)})
}

func (a *app) smsCode(c *gin.Context) {
	phone := getParam(c, "phone")
	randCode := strconv.Itoa(1000 + int(time.Now().UnixNano()%9000))

	affected, err := a.exec(`UPDATE tp_user SET SMSCode=? WHERE phonenumber=?`, randCode, phone)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": randCode})
}

func (a *app) phoneLogin(c *gin.Context) {
	phone := getParam(c, "phone")
	smsCode := getParam(c, "SMSCode")
	row, err := a.queryOne(`SELECT id FROM tp_user WHERE phonenumber=? AND SMSCode=? AND status=1`, phone, smsCode)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "token": a.generateToken(phone)})
}

func (a *app) register(c *gin.Context) {
	now := time.Now().Unix()
	_, err := a.exec(`INSERT INTO tp_user (userName,nickName,passWord,avatar,phonenumber,sex,email,idCard,address,introduction,createTime)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		getParam(c, "userName"), getParam(c, "nickName"), getParam(c, "passWord"), getParam(c, "avatar"),
		getParam(c, "phonenumber"), getParam(c, "sex"), getParam(c, "email"), getParam(c, "idCard"),
		getParam(c, "address"), getParam(c, "introduction"), now)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getUserInfo(c *gin.Context) {
	uid := mustUID(c)
	row, err := a.getUserByUID(uid)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) updateUserInfo(c *gin.Context) {
	uid := mustUID(c)
	affected, err := a.exec(`UPDATE tp_user SET avatar=?,phonenumber=?,sex=?,email=?,idCard=?,address=?,introduction=?,updateTime=?
		WHERE status=1 AND (userName=? OR phonenumber=?)`,
		getParam(c, "avatar"), getParam(c, "phonenumber"), getParam(c, "sex"), getParam(c, "email"), getParam(c, "idCard"),
		getParam(c, "address"), getParam(c, "introduction"), time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) resetPwd(c *gin.Context) {
	uid := mustUID(c)
	affected, err := a.exec(`UPDATE tp_user SET passWord=?,updateTime=? WHERE status=1 AND (userName=? OR phonenumber=?)`,
		getParam(c, "newPassword"), time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) resetName(c *gin.Context) {
	uid := mustUID(c)
	affected, err := a.exec(`UPDATE tp_user SET nickName=?,updateTime=? WHERE status=1 AND (userName=? OR phonenumber=?)`,
		getParam(c, "newName"), time.Now().Unix(), uid, uid)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getContactInfo(c *gin.Context) {
	uid := mustUID(c)
	id, err := a.getUserID(uid)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	row, err := a.queryOne(`SELECT relationship,telephone,alternatePhone,createTime FROM tp_contact WHERE status=1 AND uId=?`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) updateContactInfo(c *gin.Context) {
	uid := mustUID(c)
	id, err := a.getUserID(uid)
	if err != nil {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	affected, err := a.exec(`UPDATE tp_contact SET relationship=?,telephone=?,alternatePhone=?,updateTime=? WHERE status=1 AND uId=?`,
		getParam(c, "relationship"), getParam(c, "telephone"), getParam(c, "alternatePhone"), time.Now().Unix(), id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功"})
}

func (a *app) getBannerList(c *gin.Context) {
	typeParam := getParam(c, "type")
	rows, err := a.queryRows(`SELECT id,advTitle,advImg,type FROM tp_banner WHERE type=? AND status=1`, typeParam)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	arr := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		arr = append(arr, map[string]any{
			"id":     row["id"],
			"title":  row["advTitle"],
			"imgUrl": a.httpURL + "/static/image/" + toString(row["advImg"]),
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getNoticeInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT a.id,noticeTitle,noticeStatus,contentNotice,releaseUnit,phone,createTime,noticeName,expressId
		FROM tp_notice a JOIN tp_noticetype b ON a.expressId=b.id WHERE a.id=? AND a.status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) readNotice(c *gin.Context) {
	id := c.Param("id")
	affected, err := a.exec(`UPDATE tp_notice SET noticeStatus=1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 400, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getCategoryList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,categoryName FROM tp_category WHERE status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withNewsCover(a.httpURL, rows), "total": len(rows)})
}

func (a *app) getNewsAllList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT a.id,categoryId,categoryName,title,subTitle,content,cover,publishDate,tags,hot,commentNum,likeNum,readNum,updateTime,createTime,remark,b.appType,top,createBy
		FROM tp_news a JOIN tp_category b ON a.categoryId=b.id
		WHERE a.status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withNewsCover(a.httpURL, rows), "total": len(rows)})
}

func (a *app) getNewsInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT a.id,categoryId,categoryName,title,subTitle,content,cover,publishDate,tags,hot,commentNum,likeNum,readNum,updateTime,createTime,remark,b.appType,top,createBy
		FROM tp_news a JOIN tp_category b ON a.categoryId=b.id WHERE a.id=? AND a.status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	row["cover"] = a.httpURL + "/static/image/" + toString(row["cover"])
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": row})
}

func (a *app) likeNews(c *gin.Context) {
	id := getParam(c, "id")
	affected, err := a.exec(`UPDATE tp_news SET likeNum=likeNum+1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) likeComment(c *gin.Context) {
	id := c.Param("id")
	affected, err := a.exec(`UPDATE tp_comment SET likeNum=likeNum+1 WHERE id=? AND status=1`, id)
	if err != nil || affected == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
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
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.httpURL, rows), "total": len(rows)})
}

func (a *app) getActivityInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT id,category,title,picPath,startDate,endDate,sponsor,content,position,signUpNum,maxNum,signUpEndDate,isTop FROM tp_activity WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	row["picPath"] = a.httpURL + "/static/image/" + toString(row["picPath"])
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
		respond(c, map[string]any{"code": 403, "msg": "请求无数据"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.httpURL, rows), "total": len(rows)})
}

func (a *app) activityList(c *gin.Context, sqlText string, defaultPageSize int) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", defaultPageSize)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(sqlText, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": withActivityPic(a.httpURL, rows), "total": len(rows)})
}

func (a *app) getCourseList(c *gin.Context) {
	pageNum := intParam(c, "pageNum", 1)
	pageSize := intParam(c, "pageSize", 10)
	offset := (pageNum - 1) * pageSize

	rows, err := a.queryRows(`SELECT id,title,content,cover,video,level,duration,collection,progress FROM tp_course WHERE status=1 LIMIT ?,?`, offset, pageSize)
	if err != nil || len(rows) == 0 {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	for i := range rows {
		rows[i]["cover"] = "/static/image/" + toString(rows[i]["cover"])
	}
	respond(c, map[string]any{"code": 200, "msg": "请求成功", "data": rows, "total": len(rows)})
}

func (a *app) getCourseInfo(c *gin.Context) {
	id := c.Param("id")
	row, err := a.queryOne(`SELECT id,title,content,cover,video,level,duration,collection,progress FROM tp_course WHERE id=? AND status=1`, id)
	if err != nil || row == nil {
		respond(c, map[string]any{"code": 403, "msg": "请求失败"})
		return
	}
	chapterData, ok := a.getChapterListForCourse(mustUID(c), id)
	chapters := []map[string]any{}
	if ok {
		chapters = chapterData
	}
	row["cover"] = "/static/image/" + toString(row["cover"])
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

func getJSONParam(c *gin.Context, key string) string {
	cached, ok := c.Get("json_body")
	if !ok {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil || len(body) == 0 {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
			c.Set("json_body", map[string]any{})
			return ""
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		m := map[string]any{}
		if err = json.Unmarshal(body, &m); err != nil {
			m = map[string]any{}
		}
		c.Set("json_body", m)
		cached = m
	}
	m, ok := cached.(map[string]any)
	if !ok {
		return ""
	}
	if v, exists := m[key]; exists && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
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
