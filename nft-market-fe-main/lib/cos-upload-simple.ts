import axios from "axios";

// 这是一个简化版的上传方法，不使用 COS SDK
// 如果 SDK 有问题，可以使用这个备用方案

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
    const response = await axios.post("/api/v1/upload/cos-token", {
      file_type: fileType,
      file_name: fileName,
      file_size: fileSize,
    });

    console.log("COS Token Response:", response.data);

    // 处理不同的响应格式
    if (response.data && response.data.data && response.data.data.result) {
      return response.data.data;
    }

    if (response.data && response.data.result) {
      return response.data;
    }

    if (response.data && response.data.credentials) {
      return { result: response.data };
    }

    throw new Error("无法识别的 COS Token 响应格式");
  } catch (error: any) {
    console.error("获取 COS Token 失败:", error);
    throw new Error(`获取上传凭证失败: ${error.message}`);
  }
}

// 使用 PUT 方法直接上传（简化版）
export async function uploadFileToCOSSimple(
  file: File,
  tokenData: COSTokenResponse["result"]
): Promise<string> {
  try {
    console.log("使用简化方法上传文件...");
    console.log("上传参数:", {
      bucket: tokenData.bucket,
      region: tokenData.region,
      key: tokenData.key,
      fileName: file.name,
      fileSize: file.size,
    });

    // 构建完整的上传 URL
    const uploadUrl = `${tokenData.uploadUrl}/${tokenData.key}`;

    console.log("上传 URL:", uploadUrl);

    // 使用 PUT 方法上传文件，带上临时凭证的 security token
    const response = await axios.put(uploadUrl, file, {
      headers: {
        "Content-Type": file.type || "application/octet-stream",
        "x-cos-security-token": tokenData.credentials.sessionToken,
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          const percent = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          );
          console.log(`上传进度: ${percent}%`);
        }
      },
    });

    console.log("上传响应:", response.status, response.statusText);

    if (response.status === 200 || response.status === 204) {
      const fileUrl = `${tokenData.uploadUrl}/${tokenData.key}`;
      console.log("上传成功，文件 URL:", fileUrl);
      return fileUrl;
    } else {
      throw new Error(`上传失败: ${response.status} ${response.statusText}`);
    }
  } catch (error: any) {
    console.error("上传失败:", error);
    if (error.response) {
      console.error("响应状态:", error.response.status);
      console.error("响应数据:", error.response.data);
      console.error("响应头:", error.response.headers);
    }
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

// 上传 Metadata JSON
export async function uploadMetadataToCOSSimple(
  metadata: NFTMetadata
): Promise<string> {
  try {
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

    const tokenData = await getCOSToken(
      "document",
      metadataFile.name,
      metadataFile.size
    );

    return await uploadFileToCOSSimple(metadataFile, tokenData.result);
  } catch (error: any) {
    console.error("上传 Metadata 失败:", error);
    throw new Error(`Metadata 上传失败: ${error.message}`);
  }
}

