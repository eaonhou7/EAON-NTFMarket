"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { getCOSToken } from "@/lib/cos-upload-post";

/**
 * 测试临时凭证的有效性
 */
export function TestCredentials() {
  const [result, setResult] = useState<string>("");
  const [loading, setLoading] = useState(false);

  const testCredentials = async () => {
    setLoading(true);
    setResult("正在测试...");

    try {
      // 获取临时凭证
      const tokenData = await getCOSToken("image", "test.jpg", 1024);

      console.log("完整的 Token 数据:", tokenData);

      // 验证凭证格式
      const { result: data } = tokenData;
      const { credentials, bucket, region, key, startTime, expiredTime } = data;

      const checks = {
        "✅ tmpSecretId 存在": !!credentials.tmpSecretId,
        "✅ tmpSecretId 格式": credentials.tmpSecretId?.startsWith("AKID"),
        "✅ tmpSecretKey 存在": !!credentials.tmpSecretKey,
        "✅ sessionToken 存在": !!credentials.sessionToken,
        "✅ bucket 存在": !!bucket,
        "✅ region 存在": !!region,
        "✅ key 存在": !!key,
        "✅ startTime 存在": !!startTime,
        "✅ expiredTime 存在": !!expiredTime,
      };

      // 计算时间
      const startTimeStamp = Math.floor(
        new Date(startTime).getTime() / 1000
      );
      const currentTimeStamp = Math.floor(Date.now() / 1000);
      const timeValid = currentTimeStamp >= startTimeStamp && currentTimeStamp < expiredTime;

      setResult(`
📋 临时凭证验证结果:

${Object.entries(checks)
          .map(([key, value]) => `${value ? "✅" : "❌"} ${key}`)
          .join("\n")}

⏰ 时间验证:
- Start Time: ${startTime}
- Start Timestamp: ${startTimeStamp}
- Expired Timestamp: ${expiredTime}
- Current Timestamp: ${currentTimeStamp}
- 时间有效: ${timeValid ? "✅ 是" : "❌ 否"}
- 剩余时间: ${Math.floor((expiredTime - currentTimeStamp) / 60)} 分钟

🔑 凭证信息:
- TmpSecretId: ${credentials.tmpSecretId}
- TmpSecretKey: ${credentials.tmpSecretKey?.substring(0, 15)}...
- SessionToken: ${credentials.sessionToken?.substring(0, 30)}...

📦 存储信息:
- Bucket: ${bucket}
- Region: ${region}
- Key: ${key}

${!timeValid ? "\n⚠️ 警告: 临时凭证时间无效！" : ""}
${!checks["✅ tmpSecretId 格式"] ? "\n⚠️ 警告: tmpSecretId 格式不正确！" : ""}
      `);
    } catch (error: any) {
      console.error("测试失败:", error);
      setResult(`❌ 测试失败:\n\n${error.message}\n\n请查看控制台获取详细信息。`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 bg-card border border-border rounded-lg mb-4">
      <h3 className="text-xl font-bold mb-4">🔍 临时凭证测试工具</h3>
      <p className="text-muted-foreground mb-4">
        点击按钮测试后端返回的临时凭证是否有效
      </p>

      <Button onClick={testCredentials} disabled={loading} className="mb-4">
        {loading ? "测试中..." : "测试临时凭证"}
      </Button>

      {result && (
        <pre className="bg-background p-4 rounded border border-border overflow-auto text-xs whitespace-pre-wrap">
          {result}
        </pre>
      )}
    </div>
  );
}

