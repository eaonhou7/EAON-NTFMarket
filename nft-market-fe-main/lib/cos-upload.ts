import axios from "axios";
// import COS from "cos-js-sdk-v5"; // Removed top-level import

// COS 临时凭证响应接口
export interface COSTokenResponse {
  result: {
    credentials: {
      tmpSecretId: string;
      tmpSecretKey: string;
      sessionToken: string;
    };
    requestId: string;
    expiration: string;
    startTime: string;
    expiredTime: number;
    bucket: string;
    region: string;
    allowPrefix: string;
    allowActions: string[];
    uploadUrl: string;
    key: string;
  };
}

// 获取 COS 临时访问凭证
export async function getCOSToken(
  fileType: "image" | "video" | "audio" | "document",
  fileName: string,
  fileSize: number
): Promise<COSTokenResponse> {
  try {
    const response = await axios.post(
      "/api/v1/upload/cos-token",
      {
        file_type: fileType,
        file_name: fileName,
        file_size: fileSize,
      }
    );

    console.log("COS Token Response:", response.data);

    // 处理不同的响应格式
    // 格式 1: { data: { result: {...} } } - 实际的后端格式
    if (response.data && response.data.data && response.data.data.result) {
      console.log("使用格式 1: data.data.result");
      return response.data.data;
    }

    // 格式 2: { result: {...} }
    if (response.data && response.data.result) {
      console.log("使用格式 2: data.result");
      return response.data;
    }

    // 格式 3: 数据本身就是 result 对象（可能被拦截器处理过）
    if (response.data && response.data.credentials) {
      console.log("使用格式 3: data 本身");
      return { result: response.data };
    }

    console.log("Invalid COS token response format", response);
    console.log("请检查 API 响应格式是否正确");

    throw new Error("无法识别的 COS Token 响应格式");
  } catch (error: any) {
    console.error("获取 COS Token 失败:", error);
    throw new Error(`获取上传凭证失败: ${error.message}`);
  }
}

// 上传文件到 COS
export async function uploadFileToCOS(
  file: File,
  tokenData: COSTokenResponse["result"]
): Promise<string> {
  try {
    if (!tokenData) {
      throw new Error("Token data is undefined");
    }

    console.log("Upload token data:", tokenData);

    if (!tokenData.key || !tokenData.bucket || !tokenData.region || !tokenData.credentials) {
      throw new Error("Invalid token data structure");
    }

    // 打印临时凭证信息用于调试
    console.log("临时凭证信息:", {
      tmpSecretId: tokenData.credentials.tmpSecretId,
      tmpSecretKey: tokenData.credentials.tmpSecretKey?.substring(0, 10) + "...",
      sessionToken: tokenData.credentials.sessionToken?.substring(0, 20) + "...",
      startTime: tokenData.startTime,
      expiredTime: tokenData.expiredTime,
    });

    // 计算时间戳
    const startTimeStamp = Math.floor(new Date(tokenData.startTime).getTime() / 1000);
    const expiredTimeStamp = tokenData.expiredTime;

    console.log("时间戳:", {
      startTimeStamp,
      expiredTimeStamp,
      currentTime: Math.floor(Date.now() / 1000),
    });

    // 创建 COS 实例，使用临时凭证
    // 动态导入 COS SDK 以避免 SSR 问题
    const { default: COS } = await import("cos-js-sdk-v5");
    const cos = new COS({
      getAuthorization: (_options: any, callback: any) => {
        const credentials = {
          TmpSecretId: tokenData.credentials.tmpSecretId,
          TmpSecretKey: tokenData.credentials.tmpSecretKey,
          SecurityToken: tokenData.credentials.sessionToken,
          StartTime: startTimeStamp,
          ExpiredTime: expiredTimeStamp,
        };
        console.log("传递给 COS SDK 的凭证:", {
          TmpSecretId: credentials.TmpSecretId,
          TmpSecretKey: credentials.TmpSecretKey?.substring(0, 10) + "...",
          SecurityToken: credentials.SecurityToken?.substring(0, 20) + "...",
          StartTime: credentials.StartTime,
          ExpiredTime: credentials.ExpiredTime,
        });
        callback(credentials);
      },
    });

    // 上传文件
    return new Promise((resolve, reject) => {
      console.log("开始上传文件到 COS...");
      console.log("上传参数:", {
        Bucket: tokenData.bucket,
        Region: tokenData.region,
        Key: tokenData.key,
        FileName: file.name,
        FileSize: file.size,
      });

      cos.putObject(
        {
          Bucket: tokenData.bucket,
          Region: tokenData.region,
          Key: tokenData.key,
          Body: file,
          onProgress: (progressData: any) => {
            const percent = Math.round(progressData.percent * 100);
            console.log(`上传进度: ${percent}%`);
          },
        },
        (err: any, data: any) => {
          if (err) {
            console.error("COS 上传失败:", err);
            console.error("错误详情:", {
              code: err.code,
              message: err.message,
              statusCode: err.statusCode,
              headers: err.headers,
            });
            reject(new Error(`文件上传失败: ${err.message || "未知错误"}`));
          } else {
            console.log("COS 上传成功:", data);
            // 返回文件的访问 URL
            const fileUrl = `${tokenData.uploadUrl}/${tokenData.key}`;
            console.log("文件访问 URL:", fileUrl);
            resolve(fileUrl);
          }
        }
      );
    });
  } catch (error: any) {
    console.error("上传文件到 COS 失败:", error);
    throw new Error(`文件上传失败: ${error.message}`);
  }
}

// NFT Metadata 接口
export interface NFTMetadata {
  name: string;
  description: string;
  image: string;
  attributes?: Array<{
    trait_type: string;
    value: string | number;
  }>;
}

// 上传 Metadata JSON 到 COS
export async function uploadMetadataToCOS(
  metadata: NFTMetadata
): Promise<string> {
  try {
    // 将 metadata 转换为 JSON 字符串
    const metadataJson = JSON.stringify(metadata, null, 2);
    const metadataBlob = new Blob([metadataJson], {
      type: "application/json",
    });
    const metadataFile = new File(
      [metadataBlob],
      `metadata_${Date.now()}.json`,
      {
        type: "application/json",
      }
    );

    // 获取上传凭证
    const tokenData = await getCOSToken(
      "document",
      metadataFile.name,
      metadataFile.size
    );

    console.log("Metadata token data:", tokenData);

    // 上传 metadata
    return await uploadFileToCOS(metadataFile, tokenData.result);
  } catch (error: any) {
    console.error("上传 Metadata 失败:", error);
    throw new Error(`Metadata 上传失败: ${error.message}`);
  }
}

