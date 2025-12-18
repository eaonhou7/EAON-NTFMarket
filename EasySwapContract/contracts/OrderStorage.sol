// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {
    Initializable
} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

// import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";

import {RedBlackTreeLibrary, Price} from "./libraries/RedBlackTreeLibrary.sol";
import {LibOrder, OrderKey} from "./libraries/LibOrder.sol";

error CannotInsertDuplicateOrder(OrderKey orderKey);

contract OrderStorage is Initializable {
    using RedBlackTreeLibrary for RedBlackTreeLibrary.Tree;

    /// @dev 存储所有订单的详细信息
    /// Key: 订单哈希 (OrderKey)
    /// Value: DBOrder 结构体，包含订单详情和链表指针
    mapping(OrderKey => LibOrder.DBOrder) public orders;

    /// @dev 价格树：使用红黑树存储价格，实现 O(log n) 的插入、删除和查找
    /// 第一层 Key: NFT合约地址 (collection)
    /// 第二层 Key: 买卖方向 (Side) - Bid(买) 或 List(卖)
    /// Value: 红黑树 (RedBlackTreeLibrary.Tree)
    //    - 对于 Bid (买单)，价格按从高到低排序 (Max Heap 逻辑)
    //    - 对于 List (卖单)，价格按从低到高排序 (Min Heap 逻辑)
    mapping(address => mapping(LibOrder.Side => RedBlackTreeLibrary.Tree))
        public priceTrees;

    /// @dev 订单队列：用于解决相同价格的订单排序问题
    /// 双向链表，将相同价格的订单串联起来，遵循先入先出 (FIFO) 原则
    /// 第一层 Key: NFT合约地址 (collection)
    /// 第二层 Key: 买卖方向 (Side)
    /// 第三层 Key: 价格 (Price)
    /// Value: 订单队列结构体 (OrderQueue)，包含头尾指针 (Head, Tail)
    mapping(address => mapping(LibOrder.Side => mapping(Price => LibOrder.OrderQueue)))
        public orderQueues;

    function __OrderStorage_init() internal onlyInitializing {}

    function __OrderStorage_init_unchained() internal onlyInitializing {}

    // 辅助函数：使用 unchecked 关键字进行加法运算，节省 gas (防止溢出检查)
    // @param x: 输入值
    // @return: x + 1
    function onePlus(uint256 x) internal pure returns (uint256) {
        unchecked {
            return 1 + x;
        }
    }

    // 获取当前市场上的最优价格
    // @param collection: NFT 合约地址
    // @param side: 买卖方向
    // @return price: 最优价格
    function getBestPrice(
        address collection,
        LibOrder.Side side
    ) public view returns (Price price) {
        // RedBlackTree 自动排序
        // 如果是 Bid (买单)，我们需要最高的买价，即树的最后一个节点 (last)
        // 如果是 List (卖单)，我们需要最低的卖价，即树的第一个节点 (first)
        price = (side == LibOrder.Side.Bid)
            ? priceTrees[collection][side].last()
            : priceTrees[collection][side].first();
    }

    // 获取比给定价格差一点的下一个最优价格 (用于遍历)
    // @param collection: NFT 合约地址
    // @param side: 买卖方向
    // @param price: 当前价格
    // @return nextBestPrice: 下一个最优价格
    function getNextBestPrice(
        address collection,
        LibOrder.Side side,
        Price price
    ) public view returns (Price nextBestPrice) {
        if (RedBlackTreeLibrary.isEmpty(price)) {
            // 如果传入的价格为空 (可能是第一次查询)，则返回当前的最优价格
            nextBestPrice = (side == LibOrder.Side.Bid)
                ? priceTrees[collection][side].last()
                : priceTrees[collection][side].first();
        } else {
            // 否则，在红黑树中查找相邻节点
            // Bid: 找前一个节点 (prev)，即价格稍低一点的
            // List: 找后一个节点 (next)，即价格稍高一点的
            nextBestPrice = (side == LibOrder.Side.Bid)
                ? priceTrees[collection][side].prev(price)
                : priceTrees[collection][side].next(price);
        }
    }

    // 将新订单添加到订单簿 (存储到 orders 和更新数据结构)
    // @param order: 订单详情
    // @return orderKey: 新生成的订单哈希
    function _addOrder(
        LibOrder.Order memory order
    ) internal returns (OrderKey orderKey) {
        // 计算订单哈希
        orderKey = LibOrder.hash(order);

        // 校验: 确保订单未存在 (通过检查 maker 是否为零地址)
        if (orders[orderKey].order.maker != address(0)) {
            revert CannotInsertDuplicateOrder(orderKey);
        }

        // 步骤 1: 将价格插入红黑树 (如果该价格尚不存在)
        RedBlackTreeLibrary.Tree storage priceTree = priceTrees[
            order.nft.collection
        ][order.side];
        if (!priceTree.exists(order.price)) {
            priceTree.insert(order.price);
        }

        // 步骤 2: 将订单插入对应的价格队列 (链表操作)
        LibOrder.OrderQueue storage orderQueue = orderQueues[
            order.nft.collection
        ][order.side][order.price];

        if (LibOrder.isSentinel(orderQueue.head)) {
            // 情况 A: 该价格的队列尚未初始化 (头指针为哨兵)
            // 初始化队列，头尾指针都指向新订单之前的一个虚拟哨兵状态(这里逻辑稍微特殊)
            // 实际上是为了初始化结构，第一次插入时，队列为空

            // 更正逻辑理解：如果队列未初始化，直接创建新的 OrderQueue
            orderQueues[order.nft.collection][order.side][
                order.price
            ] = LibOrder.OrderQueue(
                LibOrder.ORDERKEY_SENTINEL,
                LibOrder.ORDERKEY_SENTINEL
            );
            // 重新获取 storage 引用
            orderQueue = orderQueues[order.nft.collection][order.side][
                order.price
            ];
        }

        if (LibOrder.isSentinel(orderQueue.tail)) {
            // 情况 B: 队列为空 (Tail 指向哨兵)
            // 头尾指针都指向当前的新订单
            orderQueue.head = orderKey;
            orderQueue.tail = orderKey;

            // 存储订单数据，Next 指针指向哨兵 (表示链表末尾)
            orders[orderKey] = LibOrder.DBOrder(
                order,
                LibOrder.ORDERKEY_SENTINEL
            );
        } else {
            // 情况 C: 队列不为空
            // 1. 将旧的尾部订单的 Next 指向新订单
            orders[orderQueue.tail].next = orderKey;

            // 2. 存储新订单数据，Next 指向哨兵
            orders[orderKey] = LibOrder.DBOrder(
                order,
                LibOrder.ORDERKEY_SENTINEL
            );

            // 3. 更新队列的 Tail 指针为新订单
            orderQueue.tail = orderKey;
        }
    }

    // 从订单簿中移除订单
    // @param order: 要移除的订单详情
    // @return orderKey: 被移除的订单哈希
    function _removeOrder(
        LibOrder.Order memory order
    ) internal returns (OrderKey orderKey) {
        // 定位到对应的订单队列
        LibOrder.OrderQueue storage orderQueue = orderQueues[
            order.nft.collection
        ][order.side][order.price];

        // 从队列头部开始遍历查找目标订单
        orderKey = orderQueue.head;
        OrderKey prevOrderKey; // 记录前一个节点的 Key，用于链表删除操作
        bool found;

        // 遍历链表直到找到订单或到达末尾 (哨兵)
        while (LibOrder.isNotSentinel(orderKey) && !found) {
            LibOrder.DBOrder memory dbOrder = orders[orderKey];

            // 比较订单的所有关键字段以确保匹配
            if (
                (dbOrder.order.maker == order.maker) &&
                (dbOrder.order.saleKind == order.saleKind) &&
                (dbOrder.order.expiry == order.expiry) &&
                (dbOrder.order.salt == order.salt) &&
                (dbOrder.order.nft.tokenId == order.nft.tokenId) &&
                (dbOrder.order.nft.amount == order.nft.amount)
            ) {
                // 找到了目标订单
                OrderKey temp = orderKey;

                // 鏈表刪除操作:
                if (
                    OrderKey.unwrap(orderQueue.head) ==
                    OrderKey.unwrap(orderKey)
                ) {
                    // 如果是头节点 (Head)，直接更新 Head 指向下一个节点
                    orderQueue.head = dbOrder.next;
                } else {
                    // 如果是中间或尾部节点，将前一个节点的 Next 指向当前节点的 Next
                    orders[prevOrderKey].next = dbOrder.next;
                }

                if (
                    OrderKey.unwrap(orderQueue.tail) ==
                    OrderKey.unwrap(orderKey)
                ) {
                    // 如果是尾节点 (Tail)，更新 Tail 指向前一个节点
                    // 注意: 如果只有一个节点，Head 更新为 Sentinel，Tail 更新为 prev (此时 prev 为 default/zero? 需确认上下文)
                    // 实际上 prevOrderKey 初始值为 0 (Sentinel 也是包装的 0)，逻辑是自洽的
                    orderQueue.tail = prevOrderKey;
                }

                // 继续迭代变量更新 (虽然已经 break)
                prevOrderKey = orderKey;
                orderKey = dbOrder.next;

                // 从存储中彻底删除该订单数据
                delete orders[temp];
                found = true;
            } else {
                // 未找到，继续遍历下一个
                prevOrderKey = orderKey;
                orderKey = dbOrder.next;
            }
        }

        if (found) {
            // 如果删除后队列为空 (Head 变为哨兵)
            if (LibOrder.isSentinel(orderQueue.head)) {
                // 删除该价格的队列 mapping 入口 (退还 gas)
                delete orderQueues[order.nft.collection][order.side][
                    order.price
                ];

                // 从红黑树中移除该价格节点
                RedBlackTreeLibrary.Tree storage priceTree = priceTrees[
                    order.nft.collection
                ][order.side];
                if (priceTree.exists(order.price)) {
                    priceTree.remove(order.price);
                }
            }
        } else {
            // 如果遍历完整个队列都没找到该订单，抛出异常
            revert("Cannot remove missing order");
        }
    }

    /**
     * @dev 批量获取订单列表 (支持分页和过滤)
     * @param collection NFT 合约地址
     * @param tokenId NFT Token ID (针对 Item Offer)
     * @param side 买卖方向
     * @param saleKind 销售类型 (单品或全系列)
     * @param count 请求的订单数量
     * @param price 起始价格 (用于分页，若为 0 则从最优价开始)
     * @param firstOrderKey 起始订单哈希 (用于遍历同一价格的队列)
     * @return resultOrders 结果订单数组
     * @return nextOrderKey 下一次查询的锚点 Key (虽然这里没用到?)
     */
    function getOrders(
        address collection,
        uint256 tokenId,
        LibOrder.Side side,
        LibOrder.SaleKind saleKind,
        uint256 count,
        Price price,
        OrderKey firstOrderKey
    )
        external
        view
        returns (LibOrder.Order[] memory resultOrders, OrderKey nextOrderKey)
    {
        resultOrders = new LibOrder.Order[](count);

        // 如果未指定起始价格，获取当前最优价格作为起点
        if (RedBlackTreeLibrary.isEmpty(price)) {
            price = getBestPrice(collection, side);
        } else {
            // 如果指定了 firstOrderKey 且为哨兵，意味着当前价格已遍历完，要从下一个价格开始
            if (LibOrder.isSentinel(firstOrderKey)) {
                price = getNextBestPrice(collection, side, price);
            }
        }

        uint256 i; // 已找到的订单计数

        // 外层循环：遍历价格树 (从最优价开始)
        while (RedBlackTreeLibrary.isNotEmpty(price) && i < count) {
            // 获取当前价格的订单队列
            LibOrder.OrderQueue memory orderQueue = orderQueues[collection][
                side
            ][price];
            OrderKey orderKey = orderQueue.head;

            // 如果指定了起始订单 key (用于页内跳转)，先定位到该订单
            if (LibOrder.isNotSentinel(firstOrderKey)) {
                while (
                    LibOrder.isNotSentinel(orderKey) &&
                    OrderKey.unwrap(orderKey) != OrderKey.unwrap(firstOrderKey)
                ) {
                    LibOrder.DBOrder memory order = orders[orderKey];
                    orderKey = order.next;
                }
                // 定位完成后，重置 firstOrderKey，以便后续循环直接处理
                firstOrderKey = LibOrder.ORDERKEY_SENTINEL;
            }

            // 内层循环：遍历当前价格的订单队列
            while (LibOrder.isNotSentinel(orderKey) && i < count) {
                LibOrder.DBOrder memory dbOrder = orders[orderKey];
                orderKey = dbOrder.next;

                // 过滤 1: 已过期的订单
                if (
                    (dbOrder.order.expiry != 0 &&
                        dbOrder.order.expiry < block.timestamp)
                ) {
                    continue;
                }

                // 过滤 2: 销售类型不匹配 (例如 Bid 请求 Collection Offer，但遇到 Item Offer)
                if (
                    (side == LibOrder.Side.Bid) &&
                    (saleKind == LibOrder.SaleKind.FixedPriceForCollection)
                ) {
                    if (
                        (dbOrder.order.side == LibOrder.Side.Bid) &&
                        (dbOrder.order.saleKind ==
                            LibOrder.SaleKind.FixedPriceForItem)
                    ) {
                        continue;
                    }
                }

                // 过滤 3: Collection Offer 单独指定 tokenId 的情况 (如果请求 Item Offer，需过滤掉 tokenId 不匹配的)
                if (
                    (side == LibOrder.Side.Bid) &&
                    (saleKind == LibOrder.SaleKind.FixedPriceForItem)
                ) {
                    if (
                        (dbOrder.order.side == LibOrder.Side.Bid) &&
                        (dbOrder.order.saleKind ==
                            LibOrder.SaleKind.FixedPriceForItem) &&
                        (tokenId != dbOrder.order.nft.tokenId)
                    ) {
                        continue;
                    }
                }

                // 收集有效订单
                resultOrders[i] = dbOrder.order;
                // 这里的 nextOrderKey 赋值逻辑似乎有问题？它只存了最后一个？
                // 实际上 getOrders 返回值 nextOrderKey 若用于翻页，需精确设计。
                // 当前代码仅做收集。
                nextOrderKey = dbOrder.next;
                i = onePlus(i);
            }
            // 当前价格队列遍历完或未满 count，继续找下一个价格
            price = getNextBestPrice(collection, side, price);
        }
    }

    // 获取最优的一个匹配订单 (即撮合时用于查找最佳对手单)
    // @param collection: NFT 合约
    // @param tokenId: NFT ID
    // @param side: 对手单方向
    // @param saleKind: 销售类型
    // @return orderResult: 找到的最佳订单
    function getBestOrder(
        address collection,
        uint256 tokenId,
        LibOrder.Side side,
        LibOrder.SaleKind saleKind
    ) external view returns (LibOrder.Order memory orderResult) {
        // 从最优价格开始查找
        Price price = getBestPrice(collection, side);

        // 遍历价格树直到找到有效订单或树遍历完
        while (RedBlackTreeLibrary.isNotEmpty(price)) {
            LibOrder.OrderQueue memory orderQueue = orderQueues[collection][
                side
            ][price];
            OrderKey orderKey = orderQueue.head;

            // 遍历该价格下的订单队列
            while (LibOrder.isNotSentinel(orderKey)) {
                LibOrder.DBOrder memory dbOrder = orders[orderKey];

                // 过滤逻辑 (与 getOrders 类似，确保类型和 ID 匹配)
                if (
                    (side == LibOrder.Side.Bid) &&
                    (saleKind == LibOrder.SaleKind.FixedPriceForItem)
                ) {
                    if (
                        (dbOrder.order.side == LibOrder.Side.Bid) &&
                        (dbOrder.order.saleKind ==
                            LibOrder.SaleKind.FixedPriceForItem) &&
                        (tokenId != dbOrder.order.nft.tokenId)
                    ) {
                        // 不匹配，跳过
                        orderKey = dbOrder.next;
                        continue;
                    }
                }

                if (
                    (side == LibOrder.Side.Bid) &&
                    (saleKind == LibOrder.SaleKind.FixedPriceForCollection)
                ) {
                    if (
                        (dbOrder.order.side == LibOrder.Side.Bid) &&
                        (dbOrder.order.saleKind ==
                            LibOrder.SaleKind.FixedPriceForItem)
                    ) {
                        orderKey = dbOrder.next;
                        continue;
                    }
                }

                // 检查有效期: 找到第一个未过期的有效订单即返回 (因为是 FIFO，队头最优先)
                if (
                    (dbOrder.order.expiry == 0 ||
                        dbOrder.order.expiry > block.timestamp)
                ) {
                    orderResult = dbOrder.order;
                    break;
                }
                orderKey = dbOrder.next;
            }

            // 如果找到了有效订单 (Price > 0 是简单判断方式)，则退出外层循环
            if (Price.unwrap(orderResult.price) > 0) {
                break;
            }

            // 否则继续找下一个价格
            price = getNextBestPrice(collection, side, price);
        }
    }

    uint256[50] private __gap;
}
