import axios from "axios";

// 使用 POST Object 方式上传（最兼容临时凭证的方式）

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

// 使用 POST Object 方式上传（推荐用于临时凭证）
export async function uploadFileToCOSPost(
  file: File,
  tokenData: COSTokenResponse["result"]
): Promise<string> {
  try {
    console.log("使用 POST Object 方式上传文件...");
    console.log("上传参数:", {
      bucket: tokenData.bucket,
      region: tokenData.region,
      key: tokenData.key,
      fileName: file.name,
      fileSize: file.size,
    });

    // 使用 FormData 构建 POST Object 请求
    const formData = new FormData();
    
    // 注意：key 必须在 file 之前
    formData.append("key", tokenData.key);
    formData.append("x-cos-security-token", tokenData.credentials.sessionToken);
    
    // 可选：设置 Content-Type
    if (file.type) {
      formData.append("Content-Type", file.type);
    }
    
    // file 必须是最后一个字段
    formData.append("file", file);

    console.log("上传到:", tokenData.uploadUrl);
    console.log("Key:", tokenData.key);
    console.log("Security Token:", tokenData.credentials.sessionToken.substring(0, 20) + "...");

    // POST 到 bucket 根路径
    const response = await axios.post(tokenData.uploadUrl, formData, {
      headers: {
        // 不要手动设置 Content-Type，让浏览器自动设置 multipart/form-data 边界
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          const percent = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          );
          console.log(`上传进度: ${percent}%`);
        }
      },
      // 不要让 axios 自动转换响应
      transformResponse: [(data) => data],
    });

    console.log("上传响应状态:", response.status);
    console.log("上传响应数据:", response.data);

    // POST Object 成功返回 204 No Content 或 200 OK
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
      console.error("响应头:", error.response.headers);
      console.error("响应数据:", error.response.data);
      
      // 尝试解析 XML 错误响应
      if (typeof error.response.data === 'string' && error.response.data.includes('<Error>')) {
        console.error("COS 错误响应 (XML):", error.response.data);
      }
    } else if (error.request) {
      console.error("没有收到响应:", error.request);
    } else {
      console.error("请求配置错误:", error.message);
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
export async function uploadMetadataToCOSPost(
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

    return await uploadFileToCOSPost(metadataFile, tokenData.result);
  } catch (error: any) {
    console.error("上传 Metadata 失败:", error);
    throw new Error(`Metadata 上传失败: ${error.message}`);
  }
}

