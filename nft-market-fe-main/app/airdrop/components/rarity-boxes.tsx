"use client";

export function RarityBoxes() {
  const boxes = [
    {
      type: "uncommon",
      color: "bg-orange-500",
      textColor: "text-orange-500",
      borderColor: "border-orange-500",
      img: "https://rcc-1312925360.cos.ap-shanghai.myqcloud.com/img_v3_01.gif",
    },
    {
      type: "rare",
      color: "bg-red-500",
      textColor: "text-red-500",
      borderColor: "border-red-500",
      img: "https://rcc-1312925360.cos.ap-shanghai.myqcloud.com/img_v3_02.gif",
    },
    {
      type: "legendary",
      color: "bg-purple-500",
      textColor: "text-purple-500",
      borderColor: "border-purple-500",
      img: "https://rcc-1312925360.cos.ap-shanghai.myqcloud.com/img_v3_03.gif",
    },
    {
      type: "mythical",
      color: "bg-green-500",
      textColor: "text-green-500",
      borderColor: "border-green-500",
      img: "https://rcc-1312925360.cos.ap-shanghai.myqcloud.com/img_v3_04.gif",
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-8 py-16">
      {boxes.map((box) => (
        <div key={box.type} className="flex flex-col items-center space-y-4">
          <div
            className={`
            w-40 h-40 
            perspective-box
            cursor-pointer
            ${box.borderColor} border-2
            hover:scale-105 transition-transform
          `}
          >
            <img
              src={box.img}
              alt={box.type}
              className="w-full h-full object-cover"
            />
            <div className="box-3d">
              <div
                className={`box-face box-front ${box.color} bg-opacity-20`}
              />
              <div className={`box-face box-back ${box.color} bg-opacity-20`} />
              <div
                className={`box-face box-right ${box.color} bg-opacity-20`}
              />
              <div className={`box-face box-left ${box.color} bg-opacity-20`} />
              <div className={`box-face box-top ${box.color} bg-opacity-20`} />
              <div
                className={`box-face box-bottom ${box.color} bg-opacity-20`}
              />
            </div>
          </div>
          <div className="text-center">
            <div className="uppercase font-mono tracking-wider">{box.type}</div>
            <div className={`text-2xl font-bold ${box.textColor}`}>???</div>
          </div>
        </div>
      ))}
    </div>
  );
}
