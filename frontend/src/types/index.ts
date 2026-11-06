export interface Attachment {
  id: string;
  name: string;
  size: number;
  type: string;
  url: string;
}

export interface NotificationMessage {
  id: string;
  content: string; // Markdown content
  images: string[]; // Image URLs
  attachments: Attachment[]; // File attachments
  level: 'info' | 'important' | 'urgent';
  duration: number; // In seconds, 0 = manual close
  sender?: string;
  createdAt: string;
  soundEnabled?: boolean;
  soundUrl?: string;
}

export interface AppConfig {
  className: string;
  port: number;
  publicPassword?: string;
  lanAuthRequired: boolean;
  defaultDuration: number;
  soundEnabled: boolean;
  customSound?: string;
  autoStart: boolean;
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
