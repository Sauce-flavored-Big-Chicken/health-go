智慧健康API 接口说明文档

# 1 项目说明

## 1.1 服务器地址

API 请求地址：http://192.168.1.251:8080

API 图片请求地址：http://192.168.1.251:8080追加api 返回的图片URL 地址除登录外，其他所有API 接口调用时Headers 都需要传递Authorization 数据，Authorization 值通过调用登录API 接口获取。

默认API 登录用户名：test01，密码：123456。

## 1.2 登录说明

处于安全考虑部分接口需要先登录获取授权TOKEN 信息才能调用接口功能，需要息的接口请参见每个接口

的详细说明。

## 1.3 注销说明

用户退出登录统一接口:/logout，无需参数，POST请求

## 1.4 安全认证

需要安全认证的接口需要在请求头设置认证信息，格式如下:参数:Authorization

参数值:登录获取TOKEN

## 1.5 系统默认用户

目前系统默认提供一个测试账号，可以使用此账号登录APP进行开发测试，如下：用户名：test01

密码：123456

当然，你也可以自己通过API创建自己的账号并进行开发测试。

注意：以下接口说明部分的样例数据如不特殊说明都是基于此账号下的数据。

## 1.6 表格分页

对于返回列表数据接口由于涉及到分页信息，需要传递分页参数，格式如下:参数:pageNum参数值:当前页码

参数:pageSize参数值:每页数据条数

## 1.7 系统值和返回状态码

本文档所有接口返回值类型如不特殊说明均为JSON 格式。返回状态如下:

| 状态码 | 说明 |
| --- | --- |
| 200 | 正常 |
| 500 | 系统异常 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 未找到资源 |

另外:如下字段出现在返回结果集中，属于业务辅助字段一般可以忽略，多数情况下数据内容可能为"null"。主要字段如下:

| 字段名 | 字段含义 | 备注 |
| --- | --- | --- |
| searchValue | 搜索内容 |  |
| createBy | 创建用户 |  |
| updateBy | 更新用户 |  |
| updateTime | 更新时间 |  |
| remark | 备注 |  |
| Params | 参数集合 |  |

# 1 通用接口

## 1.1 登录注册

### 1.1.1 手机登录

**接口地址：** `/prod-api/api/phone/login`

**请求方法：** `POST`

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"phone":"13411551115",
"SMSCode":"6767"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| phone | 手机号码 | body | true | string |
| SMSCode | 验证码 | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| token | string | 必须 | 返回token信息 |

### 1.1.2 用户登录

**接口地址：** `/prod-api/api/login`

**请求方法：** `POST`

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"userName":"test01",
"passWord":"123456"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| userName | 用户名 | body | true | string |
| passWord | 用户密码 | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| token | string | 必须 | 返回token信息 |

### 1.1.3 获取验证码

**接口地址：** `/prod-api/api/SMSCode`

**请求方法：** `GET`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

请求参数支持分页和排序参数参见表格分页和排序说明

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| userName | 手机号码 | query | true | string |

**请求示例：**

```text
/prod-api/api/SMSCode?phone=13411551115
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| token | string | 必须 | 返回token信息 |

### 1.1.4 用户注册

**接口地址：** `/prod-api/api/register`

**请求方法：** `POST`

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "avatar":"27e7fd58-0972-4dbf-941c-590624e6a886.png",
  "userName":"David",
  "nickName":"大卫",
  "passWord":"123456",
  "phonenumber":"15840669812",
  "sex":"0",
  "email":"David@163.com",
  "idCard":"210113199808242137",
  "address":"XXX",
  "introduction":"XXXX"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| avatar | 头像 | body | false | string |
| userName | 用户名 | body | true | string |
| nickName | 昵称 | body | false | string |
| passWord | 密码 | body | true | string |
| phonenumber | 电话号码 | body | true | string |
| sex | 性别0男1女 | body | true | string |
| email | 邮箱 | body | false | string |
| idCard | 身份证 | body | false | string |
| address | 住址 | body | false | string |
| introduction | 个人简介 | body | false | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

## 1.2 用户信息

### 1.2.1 查询个人基本信息

**接口地址：** `/prod-api/api/user/getUserInfo`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 返回用户信息数据 |
| ├id | integer(int64) | 必须 | 用户ID，兼容旧字段 |
| ├userId | integer(int64) | 必须 | 用户ID |
| ├avatar | string | 必须 | 用户头像完整URL |
| ├email | string | 必须 | 邮箱 |
| ├idCard | string | 必须 | 身份证号 |
| ├nickName | string | 必须 | 用户昵称 |
| ├phonenumber | string | 必须 | 手机号 |
| ├points | integer(int32) | 必须 | 用户积分，兼容旧字段 |
| ├score | integer(int32) | 必须 | 用户积分 |
| ├money | number | 必须 | 账户余额，兼容旧字段 |
| ├balance | number | 必须 | 账户余额 |
| ├sex | string | 必须 | 用户性别0男1女 |
| ├userName | string | 必须 | 用户名 |
| ├address | string | 必须 | 住址 |
| ├introduction | string | 必须 | 个人简介 |

### 1.2.2 修改个人基本信息

**接口地址：** `/prod-api/api/user/updateUserInfo`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

仅更新本次请求中传入的字段，未传字段保持原值

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"email":"lixl@163.com",
"idCard":"210882199807251656",
"nickName":"大卫王",
"phonenumber":"15898125461",
"sex":"0",
"address":"XXX",
"introduction":"XXXX"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| avatar | 用户头像 | body | false | string |
| email | 邮箱 | body | false | string |
| idCard | 身份证号 | body | false | string |
| nickName | 用户昵称 | body | false | string |
| phonenumber | 手机号 | body | false | string |
| sex | 用户性别0男1女 | body | false | string |
| address | 住址 | body | false | string |
| introduction | 个人简介 | body | false | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 1.2.3 修改用户密码

**接口地址：** `/prod-api/api/user/resetPwd`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"newPassword":"123789",
"oldPassword":"123456"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| newPassword | 用户新密码 | body | true | string |
| oldPassword | 用户旧密码 | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

#### 1.2.3.1 按用户名修改用户密码

**接口地址：** `/prod-api/api/user/resetPwdByUserName`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

仅校验当前请求token是否有效，不校验旧密码；可按传入userName直接更新该用户密码

**请求数据类型：**

`application/json`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| userName | 用户名 | body | true | string |
| newPassword | 用户新密码 | body | true | string |

**请求示例：**

```json
{"userName":"test01","newPassword":"654321"}
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

**响应示例：**

```json
{"code":200,"msg":"请求成功"}
```

### 1.2.4 查询联系人信息

**接口地址：** `/prod-api/api/user/getContactInfo`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 返回联系人信息 |
| ├name | string | 必须 | 姓名 |
| ├relationship | string | 必须 | 关系 |
| ├telephone | string | 必须 | 联系电话 |
| ├alternatePhone | string | 必须 | 备用电话 |

### 1.2.5 修改联系人基本信息

**接口地址：** `/prod-api/api/user/updateContactInfo`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

仅更新本次请求中传入的字段，未传字段保持原值

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"name":"张三",
"relationship":"abc",
"telephone":"2222222222",
"alternatePhone":"11111111111"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| name | 姓名 | body | false | string |
| relationship | 关系 | body | false | string |
| telephone | 联系电话 | body | false | string |
| alternatePhone | 备用电话 | body | false | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 1.2.6 修改用户名

**接口地址：** `/prod-api/api/user/resetName`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"newName":"aaaa"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| newName | 新用户名 | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 1.2.7 获取默认头像列表

**接口地址：** `/prod-api/api/user/avatarList`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 默认头像列表（数组） |
| ├id | integer(int64) | 必须 | ID |
| ├avatar | string | 必须 | 文件名 |
| ├avatarUrl | string | 必须 | URL路径 |
| total | string | 必须 | 总记录数 |

### 1.2.8 修改用户头像

**接口地址：** `/prod-api/api/user/updateUserAvatar`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
"avatar":"/static/avatar/avatar4.png"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| avatar | URL路径 | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

## 1.3 广告轮播

### 1.3.1 查询引导页及主页轮播

**接口地址：** `/prod-api/api/rotation/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

请求参数支持分页和排序参数参见表格分页和排序说明

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| type | 广告类型1引导页轮播2主页轮播 | query | true | string |
| pageNum | 页码 | query | false | string |
| pageSize | 每页数量 | query | false | string |

**请求示例：**

```text
/prod-api/api/rotation/list?pageNum=1&pageSize=8&type=2
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 广告轮播列表（数组） |
| ├id | integer(int64) | 必须 | 广告ID |
| ├title | string | 必须 | 广告标题 |
| ├imgUrl | string | 必须 | 广告图片完整URL |
| ├type | string | 必须 | 广告类型 |
| total | string | 必须 | 总记录数 |

## 1.4 新闻资讯

### 1.4.1 获取新闻分类

**接口地址：** `/prod-api/api/press/category/list`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 新闻分类实体 |
| ├id | integer(int64) | 必须 | 分类编号 |
| ├name | string | 必须 | 分类名称 |
| ├sort | integer(int32) | 必须 | 分类序号 |
| ├appType | string | 必须 | app类型 |
| total | string | 必须 | 总记录数 |

### 1.4.2 获取所有新闻列表

**接口地址：** `/prod-api/api/press/newsList`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/press/newsList?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 新闻列表（数组） |
| ├categoryId | integer(int64) | 必须 | 新闻分类ID |
| ├categoryName | string | 必须 | 新闻分类名称 |
| ├title | string | 必须 | 新闻标题 |
| ├subTitle | string | 必须 | 新闻副标题 |
| ├commentNum | integer(int64) | 必须 | 评论数 |
| ├content | string | 必须 | 新闻内容 |
| ├cover | string | 必须 | 新闻封面图片完整URL |
| ├tags | string | 必须 | 标签 |
| ├hot | string | 必须 | 是否热点，参见字典名：系统是否 |
| ├id | integer(int64) | 必须 | 新闻ID |
| ├likeNum | integer(int64) | 必须 | 点赞数 |
| ├publishDate | string(date-time) | 必须 | 发布日期 |
| ├readNum | integer(int64) | 必须 | 阅读数 |
| ├updateTime | string | 必须 | 更新时间 |
| ├createTime | string | 必须 | 创建时间 |
| ├remark | string | 必须 | 备注 |
| ├appType | string | 必须 | app类型 |
| ├top | string | 必须 | 是否推荐，参见字典名：系统是否 |
| ├createBy | string | 必须 | 创建人 |
| total | string | 必须 | 总记录数 |

### 1.4.3 获取分类新闻列表

**接口地址：** `/prod-api/api/press/category/newsList`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| pageNum | 页码 | query | false | string |
| pageSize | 每页数量 | query | false | string |
| id | 新闻分类id | query | true | string |

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/press/category/newsList?pageNum=1&pageSize=8&id=2
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 新闻列表（数组） |
| ├categoryId | integer(int64) | 必须 | 新闻分类ID |
| ├categoryName | string | 必须 | 新闻分类名称 |
| ├title | string | 必须 | 新闻标题 |
| ├subTitle | string | 必须 | 新闻副标题 |
| ├commentNum | integer(int64) | 必须 | 评论数 |
| ├content | string | 必须 | 新闻内容 |
| ├cover | string | 必须 | 新闻封面图片完整URL |
| ├tags | string | 必须 | 标签 |
| ├hot | string | 必须 | 是否热点，参见字典名：系统是否 |
| ├id | integer(int64) | 必须 | 新闻ID |
| ├likeNum | integer(int64) | 必须 | 点赞数 |
| ├publishDate | string(date-time) | 必须 | 发布日期 |
| ├readNum | integer(int64) | 必须 | 阅读数 |
| ├updateTime | string | 必须 | 更新时间 |
| ├createTime | string | 必须 | 创建时间 |
| ├remark | string | 必须 | 备注 |
| ├appType | string | 必须 | app类型 |
| ├top | string | 必须 | 是否推荐，参见字典名：系统是否 |
| ├createBy | string | 必须 | 创建人 |
| total | string | 必须 | 总记录数 |

### 1.4.4 获取新闻详细信息

**接口地址：** `/prod-api/api/press/news/{id}`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 新闻ID | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 新闻详情对象 |
| ├categoryId | integer(int64) | 必须 | 新闻分类ID |
| ├categoryName | string | 必须 | 新闻分类名称 |
| ├title | string | 必须 | 新闻标题 |
| ├subTitle | string | 必须 | 新闻副标题 |
| ├commentNum | integer(int64) | 必须 | 评论数 |
| ├content | string | 必须 | 新闻内容 |
| ├cover | string | 必须 | 新闻封面图片完整URL |
| ├tags | string | 必须 | 标签 |
| ├hot | string | 必须 | 是否热点，参见字典名：系统是否 |
| ├id | integer(int64) | 必须 | 新闻ID |
| ├likeNum | integer(int64) | 必须 | 点赞数 |
| ├publishDate | string(date-time) | 必须 | 发布日期 |
| ├readNum | integer(int64) | 必须 | 阅读数 |
| ├updateTime | string | 必须 | 更新时间 |
| ├createTime | string | 必须 | 创建时间 |
| ├remark | string | 必须 | 备注 |
| ├appType | string | 必须 | app类型 |
| ├top | string | 必须 | 是否推荐，参见字典名：系统是否 |
| ├createBy | string | 必须 | 创建人 |

### 1.4.5 新闻点赞

**接口地址：** `/prod-api/api/press/like/{id}`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 新闻id | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 1.4.6 发表新闻评论

**接口地址：** `/prod-api/api/comment/pressComment`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "content":"漫步在长江国家文化公园九江城区段，可以感受到长江的自然风光和文化底蕴交融碰撞，让人忍不住去探寻那些藏在历史背后的辉煌与灿烂。",
  "newsId":"1"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| content | 评论内容 | body | true | string |
| newsId | 新闻id | body | true | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 1.4.7 获取新闻评论列表

**接口地址：** `/prod-api/api/comment/comment/{id}`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 新闻ID | path | true | integer(int64) |
| pageNum | 页码 | query | false | string |
| pageSize | 每页数量 | query | false | string |

**请求示例：**

```text
/prod-api/api/comment/comment/{id}
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 评论列表（数组） |
| ├likeNum | integer(int64) | 必须 | 点赞数 |
| ├content | string | 必须 | 评论内容 |
| ├newsId | integer(int64) | 必须 | 新闻ID |
| ├userName | string | 必须 | 评论人用户名 |
| ├id | integer(int64) | 必须 | 评论ID |
| ├commentDate | string | 必须 | 评论时间 |
| total | string | 必须 | 总记录数 |

### 1.4.8 评论点赞

**接口地址：** `/prod-api/api/comment/like/{id}`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 评论id | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

## 1.5 文件上传

### 1.5.1 通用上传接口

**接口地址：** `/prod-api/api/common/upload`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`multipart/form-data`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| file | 上传的文件对象 | formData | true | file |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 上传结果对象 |
| ├path | string | 必须 | 文件路径 |
| ├avatar | string | 必须 | 访问路径 |
| ├size | integer(int64) | 必须 | 文件大小（字节） |
| ├name | string | 必须 | 临时文件名 |
| ├mime | string | 必须 | 文件MIME |
| ├fileName | string | 必须 | 原文件名 |
| ├materialName | string | 必须 | 素材名称 |

## 1.6 公告通知

### 1.6.1 通知列表

**接口地址：** `/prod-api/api/notice/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| pageNum | 页码 | query | false | string |
| pageSize | 每页数量 | query | false | string |
| noticeStatus | 通知状态，1已读，0未读 | query | false | string |

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/notice/list?pageNum=1&pageSize=8&noticeStatus=1
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 通知列表（数组） |
| ├id | integer(int64) | 必须 | 通知ID |
| ├noticeTitle | string | 必须 | 标题 |
| ├noticeStatus | string | 必须 | 状态，1已读，0未读 |
| ├contentNotice | string | 必须 | 内容 |
| ├releaseUnit | string | 必须 | 发布单位 |
| ├phone | string | 必须 | 手机 |
| ├createTime | string | 必须 | 时间 |
| ├expressId | string | 必须 | 通知类型id |
| ├noticeName | string | 必须 | 通知类型名称 |
| total | string | 必须 | 总记录数 |

### 1.6.2 通知列表详情

**接口地址：** `/prod-api/api/notice/{id}`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 通知ID | path | true | integer(int64) |

**请求示例：**

```text
/prod-api/api/notice/{id}
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 通知详情对象 |
| ├id | integer(int64) | 必须 | 通知ID |
| ├noticeTitle | string | 必须 | 标题 |
| ├noticeStatus | string | 必须 | 状态，1已读，0未读 |
| ├contentNotice | string | 必须 | 内容 |
| ├releaseUnit | string | 必须 | 发布单位 |
| ├phone | string | 必须 | 手机 |
| ├createTime | string | 必须 | 时间 |
| ├expressId | string | 必须 | 通知类型id |
| ├noticeName | string | 必须 | 通知类型名称 |

### 1.6.3 社区通知变已读

**接口地址：** `/prod-api/api/readNotice/1`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 通知id | path | true | integer(int64) |

**请求示例：**

```text
/prod-api/api/readNotice/1
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

# 2 智慧健康

## 2.1 友邻帖子

### 2.1.1 友邻帖子列表

**接口地址：** `/prod-api/api/friendly_neighborhood/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/friendly_neighborhood/list?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 友邻帖子列表（数组） |
| ├id | integer(int64) | 必须 | 帖子ID |
| ├publishName | string | 必须 | 发布人 |
| ├likeNum | integer | 必须 | 喜欢数量 |
| ├title | string | 必须 | 标题 |
| ├publishTime | string | 必须 | 发布时间 |
| ├publishContent | string | 必须 | 发布内容 |
| ├commentNum | integer | 必须 | 评论数 |
| ├imgUrl | string | 必须 | 图片完整URL |
| ├userImgUrl | string | 必须 | 用户头像完整URL |
| total | string | 必须 | 总记录数 |

### 2.1.2 发布友邻帖子

**接口地址：** `/prod-api/api/friendly_neighborhood/add`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "title":"周末一起打羽毛球",
  "publishContent":"周六下午社区活动中心约球，欢迎邻居们一起来。",
  "imgUrl":"/storage/uploads/news_hot.png"
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| title | 帖子标题 | body | true | string |
| publishContent | 帖子内容 | body | true | string |
| imgUrl | 配图地址 | body | false | string |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 创建结果 |
| ├id | integer(int64) | 必须 | 新建帖子ID |

### 2.1.3 友邻帖子发布评论

**接口地址：** `/prod-api/api/friendly_neighborhood/add/comment`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "content":"在中国式现代化新征程上策马扬鞭",
  "neighborhoodId":1
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| content | 回帖内容 | body | true | string |
| neighborhoodId | 友邻帖子ID | body | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 2.1.4 友邻帖子详情

**接口地址：** `/prod-api/api/friendly_neighborhood/{id}`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 友邻帖子ID | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 友邻帖子详情对象 |
| ├id | integer(int64) | 必须 | 帖子ID |
| ├publishName | string | 必须 | 发布人 |
| ├likeNum | integer | 必须 | 喜欢数量 |
| ├title | string | 必须 | 标题 |
| ├publishTime | string | 必须 | 发布时间 |
| ├publishContent | string | 必须 | 发布内容 |
| ├commentNum | integer | 必须 | 评论数 |
| ├imgUrl | string | 必须 | 图片完整URL |
| ├userImgUrl | string | 必须 | 用户头像完整URL |
| ├userComment | object（数组） | 必须 | 评论列表 |
| ├userComment[].id | integer(int64) | 必须 | 评论ID |
| ├userComment[].userName | string | 必须 | 用户名 |
| ├userComment[].userId | integer(int64) | 必须 | 用户ID |
| ├userComment[].avatar | string | 必须 | 用户头像完整URL |
| ├userComment[].content | string | 必须 | 评论内容 |
| ├userComment[].likeNum | integer | 必须 | 点赞数量 |
| ├userComment[].publishTime | string | 必须 | 创建时间 |
| ├userComment[].neighborhoodId | integer(int64) | 必须 | 友邻帖子ID |

## 2.2 社区服务

### 2.2.1 获取所有社区信息

**接口地址：** `/prod-api/api/community/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/community/list?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 社区列表（数组） |
| ├id | integer | 必须 | 社区ID |
| ├name | string | 必须 | 社区名称 |
| total | string | 必须 | 总记录数 |

### 2.2.2 社区动态列表

**接口地址：** `/prod-api/api/community/dynamic/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/community/dynamic/list?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 社区动态列表（数组） |
| ├id | integer | 必须 | 动态ID |
| ├icon | string | 必须 | 图标完整URL |
| ├title | string | 必须 | 标题 |
| ├publishTime | string | 必须 | 发布时间 |
| ├content | string | 必须 | 动态内容 |
| total | string | 必须 | 总记录数 |

### 2.2.3 社区活动列表

**接口地址：** `/prod-api/api/activity/list`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/activity/list?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 社区活动列表（数组） |
| ├id | integer | 必须 | 活动ID |
| ├category | string | 必须 | 分类，1文化；2体育；3公益；4亲子； |
| ├title | string | 必须 | 标题 |
| ├picPath | string | 必须 | 图片完整URL |
| ├startDate | string | 必须 | 活动开始时间 |
| ├endDate | string | 必须 | 活动结束时间 |
| ├sponsor | string | 必须 | 发起方 |
| ├content | string | 必须 | 详情 |
| ├position | string | 必须 | 活动地点 |
| ├signUpNum | integer | 必须 | 已报名人数 |
| ├maxNum | integer | 必须 | 最大数量 |
| ├signUpEndDate | string | 否 | 报名截止时间 |
| ├isTop | string | 必须 | 是否推荐，1推荐 |
| total | string | 必须 | 总记录数 |

### 2.2.4 社区分类活动列表

**接口地址：** `/prod-api/api/activity/category/list/{id}`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/activity/category/list/1?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 社区分类活动列表（数组） |
| ├id | integer | 必须 | 活动ID |
| ├category | string | 必须 | 分类，1文化；2体育；3公益；4亲子； |
| ├title | string | 必须 | 标题 |
| ├picPath | string | 必须 | 图片完整URL |
| ├startDate | string | 必须 | 活动开始时间 |
| ├endDate | string | 必须 | 活动结束时间 |
| ├sponsor | string | 必须 | 发起方 |
| ├content | string | 必须 | 详情 |
| ├position | string | 必须 | 活动地点 |
| ├signUpNum | integer | 必须 | 已报名人数 |
| ├maxNum | integer | 必须 | 最大数量 |
| ├signUpEndDate | string | 否 | 报名截止时间 |
| ├isTop | string | 必须 | 是否推荐，1推荐 |
| total | string | 必须 | 总记录数 |

### 2.2.5 社区活动详情

**接口地址：** `/prod-api/api/activity/{id}`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 社区活动id | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 活动实体 |
| ├id | integer | 必须 | 活动ID |
| ├category | string | 必须 | 分类，1文化；2体育；3公益；4亲子； |
| ├title | string | 必须 | 标题 |
| ├picPath | string | 必须 | 图片完整URL |
| ├startDate | string | 必须 | 活动开始时间 |
| ├endDate | string | 必须 | 活动结束时间 |
| ├sponsor | string | 必须 | 发起方 |
| ├content | string | 必须 | 详情 |
| ├position | string | 必须 | 活动地点 |
| ├signUpNum | integer | 必须 | 已报名人数 |
| ├maxNum | integer | 必须 | 最大数量 |
| ├signUpEndDate | string | 否 | 报名截止时间 |
| ├isTop | string | 必须 | 是否推荐，1推荐 |

### 2.2.6 获取课程列表

**接口地址：** `/prod-api/api/course/courseList`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/course/courseList?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 课程列表（数组） |
| ├id | integer(int64) | 必须 | 课程ID |
| ├title | string | 必须 | 标题 |
| ├content | string | 必须 | 内容 |
| ├cover | string | 必须 | 图片完整URL |
| ├video | string | 必须 | 视频 |
| ├level | string | 必须 | 等级 |
| ├duration | string | 必须 | 总时长 |
| ├progress | string | 必须 | 学习进度 |
| ├collection | string | 必须 | 是否收藏，1为收藏，反之0 |
| total | string | 必须 | 总记录数 |

### 2.2.7 获取课程详细信息

**接口地址：** `/prod-api/api/course/course/{id}`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 课程ID | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 课程实体 |
| ├id | integer(int64) | 必须 | 课程ID |
| ├title | string | 必须 | 标题 |
| ├content | string | 必须 | 内容 |
| ├cover | string | 必须 | 图片地址 |
| ├video | string | 必须 | 视频 |
| ├level | string | 必须 | 等级 |
| ├duration | string | 必须 | 总时长 |
| ├progress | string | 必须 | 学习进度 |
| ├collection | string | 必须 | 是否收藏，1为收藏，反之0 |
| ├chapter | object（数组） | 必须 | 章节 |
| ├chapter[].id | integer(int64) | 必须 | 章节ID |
| ├chapter[].name | string | 必须 | 章节名称 |
| ├chapter[].watch | string | 必须 | 是否观看，1为已经观看，0为未观看 |

### 2.2.8 推荐活动列表

**接口地址：** `/prod-api/api/activity/topList`

**请求方法：** `GET`

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

支持分页和排序参数参见表格分页和排序说明

**请求示例：**

```text
/prod-api/api/activity/topList?pageNum=1&pageSize=8
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 推荐活动列表（数组） |
| ├id | integer | 必须 | 活动ID |
| ├category | string | 必须 | 分类，1文化；2体育；3公益；4亲子； |
| ├title | string | 必须 | 标题 |
| ├picPath | string | 必须 | 图片完整URL |
| ├startDate | string | 必须 | 活动开始时间 |
| ├endDate | string | 必须 | 活动结束时间 |
| ├sponsor | string | 必须 | 发起方 |
| ├content | string | 必须 | 详情 |
| ├position | string | 必须 | 活动地点 |
| ├signUpNum | integer | 必须 | 已报名人数 |
| ├maxNum | integer | 必须 | 最大数量 |
| ├signUpEndDate | string | 否 | 报名截止时间 |
| ├isTop | string | 必须 | 是否推荐，1推荐 |
| total | string | 必须 | 总记录数 |

### 2.2.9 搜索社区活动

**接口地址：** `/prod-api/api/activity/search`

**请求方法：** `POST`

**请求数据类型：**

`application/json`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| words | 搜索关键字 | body | true | string |
| pageNum | 页码 | query | false | string |
| pageSize | 每页数量 | query | false | string |

**请求示例：**

```json
{
  "words":"活动"
}
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码 |
| msg | string | 必须 | 消息内容 |
| data | object | 必须 | 活动搜索结果（数组） |
| ├id | integer | 必须 | 活动ID |
| ├category | string | 必须 | 分类，1文化；2体育；3公益；4亲子； |
| ├title | string | 必须 | 标题 |
| ├picPath | string | 必须 | 图片完整URL |
| ├startDate | string | 必须 | 活动开始时间 |
| ├endDate | string | 必须 | 活动结束时间 |
| ├sponsor | string | 必须 | 发起方 |
| ├content | string | 必须 | 详情 |
| ├position | string | 必须 | 活动地点 |
| ├signUpNum | integer | 必须 | 已报名人数 |
| ├maxNum | integer | 必须 | 最大数量 |
| ├signUpEndDate | string | 否 | 报名截止时间 |
| ├isTop | string | 必须 | 是否推荐，1推荐 |
| total | string | 必须 | 总记录数 |

### 2.2.10 活动报名

**接口地址：** `/prod-api/api/registration`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "activityId":2
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| activityId | 活动ID | body | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 2.2.11 活动签到

**接口地址：** `/prod-api/api/checkin/{id}`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 活动ID | path | true | integer(int64) |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

### 2.2.12 活动评论

**接口地址：** `/prod-api/api/registration/comment/{id}`

**请求方法：** `PUT`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/json`

**请求示例：**

```json
{
  "evaluate":"活动组织很好，体验不错",
  "star":4
}
```

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 活动ID | path | true | integer(int64) |
| evaluate | 活动评价 | body | true | string |
| star | 评分 | body | true | integer |

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |

## 2.3 题库答题

### 2.3.1 获取题目列表

**接口地址：** `/prod-api/api/question/questionList/{id}/{level}`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| id | 模块ID | path | true | string |
| level | 难度等级 | path | true | string |
| count | 返回题目数量，默认5 | query | false | integer |

**请求示例：**

```text
/prod-api/api/question/questionList/1/1?count=10
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | array | 必须 | 题目列表 |
| ├id | integer | 必须 | 题目ID |
| ├questionType | string | 必须 | 题目类型，`1`选择题，`4`判断题 |
| ├question | string | 必须 | 题干 |
| ├optionA | string | 必须 | 选项A |
| ├optionB | string | 必须 | 选项B |
| ├optionC | string | 否 | 选项C |
| ├optionD | string | 否 | 选项D |
| ├optionE | string | 否 | 选项E |
| ├optionF | string | 否 | 选项F |
| ├answer | string | 必须 | 正确答案 |
| ├parseText | string | 必须 | 解析文本（推荐使用） |
| ├score | integer | 必须 | 题目分值 |
| total | integer | 必须 | 返回题目总数 |

### 2.3.2 提交答案

**接口地址：** `/prod-api/api/question/submit`

**请求方法：** `POST`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

接口自动判题，`score` 可不传；不传时系统按判题结果自动计算得分

**请求数据类型：**

`application/json`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| qId | 题目ID | body | true | integer |
| answer | 用户答案 | body | true | string |
| score | 本次得分（可选） | body | false | integer |

**请求示例：**

```json
{
  "qId": 5,
  "answer": "C"
}
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 提交结果 |
| ├qId | string | 必须 | 题目ID |
| ├userAnswer | string | 必须 | 用户答案 |
| ├isCorrect | integer | 必须 | 是否答对，`1`答对，`0`答错 |
| ├correctAnswer | string | 必须 | 正确答案 |
| ├score | integer | 必须 | 本次得分 |

### 2.3.3 答题统计

**接口地址：** `/prod-api/api/question/statistics`

**请求方法：** `GET`

**接口描述：**

请求头需要token参数，具体格式参见安全认证说明

默认统计 `moduleId=1`，也支持通过参数指定模块

**请求数据类型：**

`application/x-www-form-urlencoded`

**请求参数：**

| 参数名称 | 参数说明 | 请求类型 | 必须 | 数据类型 |
| --- | --- | --- | --- | --- |
| moduleId | 模块ID，默认1 | query | false | string |

**请求示例：**

```text
/prod-api/api/question/statistics?moduleId=1
```

**响应参数：**

| 名称 | 类型 | 是否必须 | 备注 |
| --- | --- | --- | --- |
| code | string | 必须 | 状态码，200正确，其他错误 |
| msg | string | 必须 | 返回消息内容 |
| data | object | 必须 | 统计结果 |
| ├answerAccuracyPercent | string | 必须 | 答题总正确率百分比 |
| ├totalAnswered | integer | 必须 | 累计答题数 |
| ├totalWrong | integer | 必须 | 累计错题数 |
| ├todayAnswered | integer | 必须 | 今日答题数 |
| ├levelProgress | array | 必须 | 各难度完成进度 |
| ├levelProgress[].level | string | 必须 | 难度等级 |
| ├levelProgress[].totalQuestions | integer | 必须 | 该等级题库总题数 |
| ├levelProgress[].completedQuestions | integer | 必须 | 该等级已完成题数（去重题目） |
| ├levelProgress[].correctCount | integer | 必须 | 该等级累计答对数 |
| ├levelProgress[].wrongCount | integer | 必须 | 该等级累计答错数 |
| ├levelProgress[].progressPercent | string | 必须 | 该等级完成进度百分比 |
