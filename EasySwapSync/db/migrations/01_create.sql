create table ob_activity_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    activity_type      tinyint                 not null comment '(1:Buy,2:Mint,3:List,4:Cancel Listing,5:Cancel Offer,6.Make Offer,7.Sell,8.Transfer,9.collection-bid,10.item-bid)',
    maker              varchar(42)             null comment '对于buy,sell,listing,transfer类型指的是nft流转的起始方，即卖方address。对于其他类型可以理解为发起方，如make offer谁发起的from就是谁的地址',
    taker              varchar(42)             null comment '目标方,和maker相对',
    marketplace_id     tinyint     default 0   not null,
    collection_address varchar(42)             null,
    token_id           varchar(128)            null,
    currency_address   varchar(42) default '1' not null comment '货币类型(1表示eth)',
    price              decimal(30) default 0   not null comment 'nft 价格',
    sell_price         decimal(30) default 0   not null comment '池子相关数据,出售价格',
    buy_price          decimal(30) default 0   not null comment '池子相关数据,购买价格',
    block_number       bigint      default 0   not null comment '区块号',
    tx_hash            varchar(66)             null comment '交易事务hash',
    event_time         bigint                  null comment '链上事件发生的时间',
    create_time        bigint                  null comment '创建时间',
    update_time        bigint                  null comment '更新时间',
    constraint index_tx_collection_token_type
        unique (tx_hash, collection_address, token_id, activity_type)
)
    collate = utf8mb4_general_ci;

create table ob_collection_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    symbol             varchar(128)         not null comment '项目标识',
    chain_id           bigint    default 1 not null comment '链类型(1:以太坊)',
    auth               tinyint    default 0 not null comment '认证(0:默认未认证1:认证通过2:认证不通过)',
    token_standard     bigint               not null comment '合约实现标准',
    name               varchar(128)         not null comment '项目名称',
    creator            varchar(42)          not null comment '创建者',
    address            varchar(42)          not null comment '链上合约地址',
    owner_amount       bigint     default 0 not null comment '拥有item人数',
    item_amount        bigint     default 0 not null comment '该项目NFT的发行总量',
    description        varchar(2048)        null comment '项目描述',
    website            varchar(512)         null comment '项目官网地址',
    twitter            varchar(512)         null comment '项目twitter地址',
    discord            varchar(512)         null comment '项目 discord 地址',
    instagram          varchar(512)         null comment '项目 instagram 地址',
    floor_price        decimal(30)          null comment '整个collection中item的最低的listing价格',
    sale_price         decimal(30)          null comment '整个collection中bid的最高的价格',
    volume_total       decimal(30)          null comment '总交易量',
    image_uri          varchar(512)         null comment '项目封面图的链接',
    banner_uri         varchar(512)         null comment 'banner image uri',
    opensea_ban_scan   tinyint    default 0 null comment '（0.未扫描 1扫描过）',
    is_syncing         tinyint(1) default 0 not null,
    history_sale_sync  tinyint    default 0 not null,
    history_overview   int        default 0 not null comment '是否生成历史overview 0:已经生成 1:等待生成 2:生成错误',
    floor_price_status int        default 0 not null,
    create_time        bigint               null comment '创建时间',
    update_time        bigint               null comment '更新时间',
    constraint index_unique_address
        unique (address)
)
    collate = utf8mb4_general_ci;

create index index_collection_token_type
    on ob_activity_sepolia (collection_address, token_id, activity_type);

create index index_hash_collection_token_type
    on ob_activity_sepolia (tx_hash, collection_address, token_id, activity_type);

create index index_tx_collection_token_type_time
    on ob_activity_sepolia (tx_hash, collection_address, token_id, activity_type, event_time);

create table ob_collection_floor_price_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    collection_address varchar(42) not null comment '链上合约地址',
    price              decimal(30) null comment 'token 价格',
    event_time         bigint      null comment '事件时间',
    create_time        bigint      null comment '创建时间',
    update_time        bigint      null comment '更新时间',
    constraint index_price
        unique (collection_address, price, event_time)
)
    collate = utf8mb4_general_ci;

create index index_collection_address
    on ob_collection_floor_price_sepolia (collection_address);

create index index_event_time
    on ob_collection_floor_price_sepolia (event_time);

create table ob_collection_import_record_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    collection_address varchar(42)              not null,
    msg                varchar(1600) default '' not null,
    finished_stage     tinyint(1)    default 0  not null comment '已完成的阶段。0表示加入任务，1表示导入collection完成，2全部完成(指item导入完成，photo不好记录不影响此处的阶段)',
    create_time        bigint                   not null,
    update_time        bigint                   not null
)
    collate = utf8mb4_general_ci;

create table ob_global_collection_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    collection_address varchar(42)       not null,
    token_standard     tinyint default 0 not null,
    import_status      tinyint default 0 not null,
    create_time        bigint            not null,
    update_time        bigint            not null
)
    collate = utf8mb4_general_ci;

create table ob_item_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    chain_id           bigint      default 1                     not null comment '链类型',
    token_id           varchar(128)                               not null comment 'token_id',
    name               varchar(128)                               not null comment 'nft名称',
    owner              varchar(42)                                null comment '拥有者',
    collection_address varchar(42)                                null comment '合约地址',
    creator            varchar(42)                                not null comment '创建者',
    supply             bigint                                     not null comment 'item最多可以有多少份',
    list_price         decimal(30)                                null comment '上架价格',
    list_time          bigint                                     null comment '上架时间',
    sale_price         decimal(30)                                null comment '销售价格',
    views              bigint                                     null comment '浏览量',
    is_opensea_banned  tinyint(1)   default 0                     null comment '是否被opensea标记禁止交易',
    create_time        bigint                                     null comment '创建时间',
    update_time        bigint                                     null comment '更新时间',
    constraint index_collection_token
        unique (collection_address, token_id)
)
    collate = utf8mb4_general_ci;

create index index_collection_item_name
    on ob_item_sepolia (collection_address, token_id, name);

create index index_collection_owner
    on ob_item_sepolia (collection_address, owner);

create index index_collection_token_owner
    on ob_item_sepolia (collection_address, token_id, owner);

create index index_owner
    on ob_item_sepolia (owner);

create table ob_item_external_sepolia
(
    id                  bigint auto_increment comment '主键'
        primary key,
    collection_address  varchar(42)             not null,
    is_uploaded_oss     tinyint(1)  default 0   null comment '是否已上传oss(0:未上传,1:已上传)',
    upload_status       tinyint     default 0   not null comment '标记上传oss状态',
    meta_data_uri       varchar(512)            null comment '元数据地址',
    image_uri           varchar(512)            null,
    oss_uri             text                    null comment '图片地址',
    token_id            varchar(128)            null,
    is_video_uploaded   tinyint(1)  default 0   null comment 'video是否已上传oss(0:未上传,1:已上传)',
    video_upload_status tinyint     default 0   not null comment '标记video上传oss状态',
    video_type          varchar(64) default '0' not null comment '标记video 类型',
    video_uri           varchar(512)            null comment 'video 原始uri',
    video_oss_uri       varchar(512)            null comment 'video oss uri',
    create_time         bigint                  null,
    update_time         bigint                  null,
    constraint index_collection_token
        unique (collection_address, token_id)
)
    collate = utf8mb4_general_ci;

create table ob_item_trait_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    collection_address varchar(42)  not null comment 'collection主键',
    token_id           varchar(128) not null comment 'item主键',
    trait              varchar(128) not null comment '属性名称',
    trait_value        varchar(512) not null comment '属性值',
    create_time        bigint       null comment '创建时间',
    update_time        bigint       null comment '更新时间'
)
    collate = utf8mb4_general_ci;

create index index_collection_token
    on ob_item_trait_sepolia (collection_address, token_id);

create index index_collection_trait_value
    on ob_item_trait_sepolia (collection_address, trait, trait_value);

create index index_trait_value
    on ob_item_trait_sepolia (trait, trait_value);

create table ob_order_sepolia
(
    id                 bigint auto_increment comment '主键'
        primary key,
    marketplace_id     tinyint     default 0   not null comment '0.locol 1.opensea 2.looks 3.x2y2',
    collection_address varchar(42)             null,
    token_id           varchar(128)            null,
    order_id           varchar(66)             not null comment '订单hash',
    order_status       tinyint     default 0   not null comment '标记订单状态',
    event_time         bigint                  null comment '订单时间',
    expire_time        bigint                  null,
    price              decimal(30) default 0   not null,
    maker              varchar(42)             null,
    taker              varchar(42)             null,
    quantity_remaining bigint      default 0   not null,
    size               bigint      default 1   not null,
    currency_address   varchar(42) default '1' not null,
    order_type         tinyint                 not null,
    salt               bigint      default 0   null,
    create_time        bigint                  null comment '创建时间',
    update_time        bigint                  null comment '更新时间',
    constraint index_hash
        unique (order_id)
)
    collate = utf8mb4_general_ci;

create index index_collection_maker_status_type_market_token_id
    on ob_order_sepolia (collection_address, maker, order_status, order_type, marketplace_id, token_id);

create index index_collection_token
    on ob_order_sepolia (collection_address, token_id);

create table ob_user
(
    id          bigint auto_increment comment '主键'
        primary key,
    address     varchar(66)          not null comment '用户地址',
    is_allowed  tinyint(1) default 0 not null comment '是否允许用户访问',
    is_signed   tinyint(1) default 0 null,
    create_time bigint               null comment '创建时间',
    update_time bigint               null comment '更新时间',
    constraint index_address
        unique (address)
)
    collate = utf8mb4_general_ci;

create table ob_indexed_status
(
    id                 bigint auto_increment comment '主键'
        primary key,
    chain_id           bigint  default 1 not null comment '链id (1:以太坊, 56: BSC)',
    last_indexed_block bigint  default 0 null comment '区块号',
    last_indexed_time  bigint            null comment '最后同步时间戳',
    index_type         tinyint default 0 not null comment '0:activity, 1:trade info, 2:listing,3:sale,4:exchange,5:floor price',
    create_time        bigint            null,
    update_time        bigint            null
)
    collate = utf8mb4_general_ci;