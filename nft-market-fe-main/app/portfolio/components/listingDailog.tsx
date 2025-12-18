import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Transition,
} from "@headlessui/react";
import { Fragment, useState } from "react";
import { ListingNftDialog } from "./listingNftDialog";


export function ListingDialog({
  open,
  close,
  collections,
  canListNfts,
}: {
  open: boolean;
  close: () => void;
  collections: any[];
  canListNfts: any[];
}) {

  

  // TODO: 知道当前用户有哪些NFT
  // 1. 哪些是可以挂单的 => a). 用户手上的NFT得从属于 后端返回的 collections 列表 b). 用户手上的NFT 没有授权给 资产管理合约

  /***
   * const collections = [...]
   * const myNfts = [...]
   * const vaultAddress = "0x..."
   * myNfts = myNfts.map(it  => {
   *  let canIList = collections.find(c => c.address === it.contractAddress) != undefined
   * let didList = fals
   *  ether.getContractAt("ERC721", it.contractAddress).then(c => {
   *    c.getApproved(it.tokenId).then(approved => {
   *      canIList = didList= approved !== vaultAddress // 如果全等，说明用户前面执行过挂单操作
   *    })
   *  })
   *  return {
   *    ...it,
   *    canIList: false, // 是否授权给 资产管理合约
   *    didList
   * }
   * })
   */

  const [nftDialogOpen, setNftDialogOpen] = useState(false);

  function closeNftDialog() {
    setNftDialogOpen(false);
  }

  function openNftDialog() {
    setNftDialogOpen(true);
  }

  return (
    <>
      <ListingNftDialog canListNfts={canListNfts} open={nftDialogOpen} close={closeNftDialog} />
      <Dialog as="div" className="relative z-50" open={open} onClose={close}>
        {/* 背景遮罩 */}
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/25" />
        </Transition.Child>

        {/* 对话框内容 */}
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <DialogPanel className="w-full max-w-md transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                <DialogTitle
                  as="h3"
                  className="text-lg font-medium leading-6 text-gray-900"
                >
                  我的挂单
                </DialogTitle>

                {/* 这里添加你的对话框内容 */}
                <div className="mt-4">
                  <p className="text-sm text-gray-500">
                    这里暂时没有挂单数据～
                  </p>
                </div>

                {/* 按钮区域 */}
                <div className="mt-6 flex justify-end space-x-3">
                  <button
                    type="button"
                    className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-900 hover:bg-gray-200"
                    onClick={close}
                  >
                    好
                  </button>
                  <button
                    type="button"
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                    onClick={openNftDialog}
                  >
                    去挂单
                  </button>
                </div>
              </DialogPanel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </>
  );
}
