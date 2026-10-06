# 背景
当前线上用户反馈会进行数据的存储，并且通过接口进行简单的展示；
页面只能看到意见反馈的记录，不能够更精确的对反馈内容进行归类，以及图表的形式去展示多维度的反馈情况；也不能够对关注的指标进行告警通知；

# 现状
意见反馈数据可以通过接口拿到；具体接口信息如下：

## 接口 CURL
意见反馈解耦 curl：
curl 'https://demo.example.com/api/feedback_list?is_ajax=1&app_id=1&version_id=&feedback_type=0&platform_id=2&to_page=3' \
  -H 'accept: application/json, text/javascript, */*; q=0.01' \
  -H 'accept-language: zh-CN,zh;q=0.9' \
  -b 'admin_id=1; MY_EVENT_PASSPORT=demo; PHPSESSID=demo; test_admin_passport=demo; isOa=1; username=demo_user; admin_passport=demo' \
  -H 'priority: u=1, i' \
  -H 'referer: https://demo.example.com/api/feedback_list?is_ajax=0&app_id=1&platform_id=2&user_mode=1,2,3' \
  -H 'sec-ch-ua: "Not/A)Brand";v="99", "Chromium";v="148"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "macOS"' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36' \
  -H 'x-requested-with: XMLHttpRequest'
## 接口数据
意见反馈接口数据
```
{
    "data": {
        "app_list": {
            "1": {
                "模式一": "0,2",
                "模式二": "1,3"
            },
            "2": "应用B",
            "7": "应用C"
        },
        "platform_list": {
            "2": "iOS",
            "3": "Android"
        },
        "param": {
            "app_id": 1,
            "type": 0,
            "next": 1
        },
        "feedback_question_list": [
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "开心大团圆",
                "content": "为什么现在红包变成0.01了，金豆双倍没有了，兑换的东西也变少变差了，支撑不起了吗？要破产了吗？",
                "image": "",
                "review_count": "0",
                "phone_mode": "CHA-AL80",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "22",
                "status": "0",
                "updated_at": "2026-05-28 23:01:41",
                "created_at": "2026-05-28 23:01:41",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.06.0",
                "review_more": false,
                "v": "10"
            },
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "3",
                "screen_name": "用户甲",
                "content": "记录突然就没有了",
                "image": "",
                "review_count": "0",
                "phone_mode": "2510DRK44C",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "11",
                "status": "0",
                "updated_at": "2026-05-28 23:00:55",
                "created_at": "2026-05-28 23:00:55",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.06.0",
                "review_more": false,
                "v": "16"
            },
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "10",
                "screen_name": "用户甲",
                "content": "广告太多了，广告非常的多，投诉你们\n",
                "image": "",
                "review_count": "0",
                "phone_mode": "iPhone18,1",
                "version_id": "0",
                "platform_id": "2",
                "channel_id": "0",
                "status": "0",
                "updated_at": "2026-05-28 22:49:35",
                "created_at": "2026-05-28 22:49:35",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.05.0",
                "review_more": false,
                "v": "26.3"
            },
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "2",
                "screen_name": "用户甲",
                "content": "手机使用的是鸿蒙系统，升级后用不了日记里面的上传照片功能？",
                "image": "",
                "review_count": "0",
                "phone_mode": "BLK-AL00",
                "version_id": "0",
                "platform_id": "8",
                "channel_id": "6666",
                "status": "0",
                "updated_at": "2026-05-28 22:35:59",
                "created_at": "2026-05-28 22:35:59",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.07.0",
                "review_more": false,
                "v": "6.1.0"
            },
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "带笑evan",
                "content": "鸿蒙版本看不了全部爱爱记录",
                "image": "",
                "review_count": "0",
                "phone_mode": "SGT-AL00",
                "version_id": "0",
                "platform_id": "8",
                "channel_id": "6666",
                "status": "0",
                "updated_at": "2026-05-28 22:27:25",
                "created_at": "2026-05-28 22:27:25",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.07.0",
                "review_more": false,
                "v": "6.1.0"
            },
            {
                "id": "20001",
                "qq": null,
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "3",
                "screen_name": "laxc869067",
                "content": "意见，喂养记录，开始时间结束时间点能否改成填写方式，翻滚方式好麻烦，我很多时候是过时后再去记录的，当下哄孩子操作事情不可能事后那么快填上，还有父母帮忙记录手写时间后我下班回来后补情况的，翻滚方式很麻烦，滚来滚去，点错了退出有重来",
                "image": [
                    "https://sc.example.com/img_10001.jpg"
                ],
                "review_count": "0",
                "phone_mode": "ALN-AL80",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "22",
                "status": "0",
                "updated_at": "2026-05-28 22:11:45",
                "created_at": "2026-05-28 22:11:45",
                "reason_id": "0",
                "mark": "",
                "file_url": "https://oss.example.com/record_demo.db\n\nhttps://oss.example.com/reduce_plan_demo.db\nhttps://oss.example.com/period_demo.db\nhttps://oss.example.com/pregnancy_demo.db\nhttps://oss.example.com/seeyou_demo.db\n",
                "version_name": "9.06.0",
                "review_more": false,
                "v": "12"
            },
            {
                "id": "20001",
                "qq": "10001",
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "",
                "content": "无法登陆",
                "image": "",
                "review_count": "0",
                "phone_mode": "V2361A",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "244",
                "status": "0",
                "updated_at": "2026-05-28 22:04:53",
                "created_at": "2026-05-28 22:04:53",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.6.0",
                "review_more": false,
                "v": "14"
            },
            {
                "id": "20001",
                "qq": "10001",
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "用户甲",
                "content": "会员项目太多",
                "image": "",
                "review_count": "0",
                "phone_mode": "V2352A",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "244",
                "status": "0",
                "updated_at": "2026-05-28 22:00:37",
                "created_at": "2026-05-28 22:00:37",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.6.0",
                "review_more": false,
                "v": "16"
            },
            {
                "id": "20001",
                "qq": "10001",
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "用户甲",
                "content": "太对收费的东西，免费的太少",
                "image": "",
                "review_count": "0",
                "phone_mode": "JER-AN20",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "22",
                "status": "0",
                "updated_at": "2026-05-28 21:57:27",
                "created_at": "2026-05-28 21:57:27",
                "reason_id": "0",
                "mark": "",
                "file_url": "",
                "version_name": "9.6.0",
                "review_more": false,
                "v": "12"
            },
            {
                "id": "20001",
                "qq": "10001",
                "app_id": "1",
                "type": "0",
                "user_id": "10001",
                "user_mode": "0",
                "screen_name": "用户甲",
                "content": "感觉不是很准",
                "image": "",
                "review_count": "0",
                "phone_mode": "22127RK46C",
                "version_id": "0",
                "platform_id": "3",
                "channel_id": "11",
                "status": "0",
                "updated_at": "2026-05-28 21:47:52",
                "created_at": "2026-05-28 21:47:52",
                "reason_id": "0",
                "mark": "",
                "file_url": "https://oss.example.com/record_demo.db,,https://oss.example.com/reduce_plan_demo.db,https://oss.example.com/period_demo.db,https://oss.example.com/pregnancy_demo.db,https://oss.example.com/seeyou_demo.db,",
                "version_name": "9.7.0",
                "review_more": false,
                "v": "16"
            }
        ],
        "feedback_review_list": {
            "20039": [],
            "20038": [],
            "20037": [],
            "20036": [],
            "20035": [],
            "20034": [],
            "20033": [],
            "20032": [],
            "20031": [],
            "20030": []
        }
    }
}
```


# 目标
现在就要要基于这个意见反馈的数据，进行更加专业的元数据管理、智能归类、反馈分析、允许自定义配置多维度的图表展示、基于图表的指标设置阈值进行告警、配置企业通讯软体的告警方式进行群机器人告警；

# 需求
1. 意见反馈管理台，支持账号密码登录；
2. 意见反馈元数据存储管理，支持基于反馈内容进行问题智能分类；可以考虑使用LLM 大模型进行意见反馈的理解分类；
3. 意见反馈管理列表，支持多字段可选条件查询，列表支持必要字段的排序；
4. 基于意见反馈元数据的多维度图表可视化pannel，构建 dashboard；类似 grafana 的效果；
5. 允许再图表中设置阈值，支持配置企业通讯APP群机器人告警，以及简单够用的告警策略和告警文案自定义配置；
6. 支持多数据源的数据拉取到意见反馈；支持可以拓展；当前需要支持：基于 API接口意见反馈数据拉取到元数据存储，增量监听新增数据同步入库；
