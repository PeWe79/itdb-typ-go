import { toast } from 'sonner';
import { apiBlob } from '@/lib/auth';

/* 下载/预览接口需要 Authorization 头，<a> 链接带不上，统一走带鉴权的 blob 通道 */
export async function downloadStoredFile(path: string, downloadName: string) {
  try {
    const blob = await apiBlob(path);
    if (!blob.size) return;
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = downloadName;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '文件下载失败');
  }
}

export async function previewStoredFile(path: string) {
  try {
    const blob = await apiBlob(path);
    if (!blob.size) return;
    const url = URL.createObjectURL(blob);
    window.open(url, '_blank');
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '文件预览失败');
  }
}
