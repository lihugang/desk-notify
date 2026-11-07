export interface Attachment {
  id: string;
  name: string;
  size: number;
  type: string;
  url: string;
  docPath?: string;
  localPath?: string;
}

export interface NotificationMessage {
  id: string;
  content: string; // Markdown content
  images: string[]; // Image URLs
  attachments: Attachment[]; // File attachments
  pinned: boolean; // 置顶显示 (true) 或 后台默默打开并闪烁任务栏 (false)
  duration: number; // In seconds, 0 = manual close
  sender?: string;
  createdAt: string;
  soundEnabled?: boolean;
  soundUrl?: string;
}

export interface AppConfig {
  className: string;
  port?: number;
  password?: string;
  hasPassword?: boolean;
  newPassword?: string;
  publicPassword?: string;
  hasPublicPassword?: boolean;
  newPublicPassword?: string;
  newAdminPassword?: string;
  lanAuthRequired: boolean;
  soundEnabled: boolean;
  customSound?: string;
  fontSize?: string;
  autoStart: boolean;
  firstRunDone?: boolean;
  documentsDir?: string;
  logsDir?: string;
  frp: {
    enabled: boolean;
    serverAddr: string;
    serverPort: number;
    token: string;
    remotePort: number;
    subdomain?: string;
  };
}

export interface SystemStatus {
  className: string;
  lanIPs: string[];
  port: number;
  frpStatus: 'connected' | 'disconnected' | 'connecting' | 'disabled';
  frpUrl?: string;
}
