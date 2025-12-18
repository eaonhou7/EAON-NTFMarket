"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import axios from "axios";

/**
 * 这是一个测试组件，用于测试上传 API 是否正常工作
 * 使用方法：在 page.tsx 中临时导入并渲染这个组件
 */
export function TestUploadAPI() {
  const [result, setResult] = useState<string>("");
  const [loading, setLoading] = useState(false);

  const testCOSTokenAPI = async () => {
    setLoading(true);
    setResult("正在测试...");
    
    try {
      console.log("发送请求到: /api/v1/upload/cos-token");
      
      const response = await axios.post("/api/v1/upload/cos-token", {
        file_type: "image",
        file_name: "test.jpg",
        file_size: 1024,
      });

      console.log("完整响应:", response);
      console.log("响应数据:", response.data);
      console.log("响应状态:", response.status);
      console.log("响应头:", response.headers);

      setResult(
        `✅ API 调用成功！\n\n` +
        `响应状态: ${response.status}\n\n` +
        `响应数据:\n${JSON.stringify(response.data, null, 2)}`
      );
    } catch (error: any) {
      console.error("API 调用失败:", error);
      
      let errorMessage = "❌ API 调用失败！\n\n";
      
      if (error.response) {
        // 服务器返回了错误响应
        errorMessage += `状态码: ${error.response.status}\n`;
        errorMessage += `响应数据: ${JSON.stringify(error.response.data, null, 2)}\n`;
      } else if (error.request) {
        // 请求已发送但没有收到响应
        errorMessage += `没有收到响应\n`;
        errorMessage += `可能的原因：\n`;
        errorMessage += `1. 后端服务未启动\n`;
        errorMessage += `2. API 地址配置错误\n`;
        errorMessage += `3. 网络问题\n`;
      } else {
        // 请求配置出错
        errorMessage += `错误: ${error.message}\n`;
      }
      
      setResult(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-8 bg-card border border-border rounded-lg">
      <h2 className="text-2xl font-bold mb-4">上传 API 测试工具</h2>
      <p className="text-muted-foreground mb-4">
        点击下方按钮测试后端上传 API 是否正常工作
      </p>
      
      <Button 
        onClick={testCOSTokenAPI} 
        disabled={loading}
        className="mb-4"
      >
        {loading ? "测试中..." : "测试 COS Token API"}
      </Button>

      {result && (
        <pre className="bg-background p-4 rounded border border-border overflow-auto text-sm whitespace-pre-wrap">
          {result}
        </pre>
      )}
      
      <div className="mt-4 p-4 bg-muted/30 rounded">
        <h3 className="font-semibold mb-2">预期的 API 响应格式：</h3>
        <pre className="text-xs overflow-auto">
{`{
  "result": {
    "credentials": {
      "tmpSecretId": "AKID...",
      "tmpSecretKey": "...",
      "sessionToken": "..."
    },
    "requestId": "...",
    "expiration": "2024-01-01T12:00:00Z",
    "startTime": "2024-01-01T11:00:00Z",
    "expiredTime": 1704110400,
    "bucket": "nft-assets",
    "region": "ap-beijing",
    "allowPrefix": "image/guest_192.168.1.100/",
    "allowActions": ["name/cos:PutObject", "name/cos:PostObject"],
    "uploadUrl": "https://nft-assets.cos.ap-beijing.myqcloud.com",
    "key": "image/guest_192.168.1.100/1704106800_a1b2c3d4.jpg"
  }
}`}
        </pre>
      </div>
    </div>
  );
}

