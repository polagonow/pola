"use client";

import { Descriptions, Button, Space, Typography } from "antd";
import { EditOutlined, ArrowLeftOutlined } from "@ant-design/icons";
import DeleteButton from "@/components/products/delete-button";

const { Title } = Typography;

export default function ProductShowView({
  item,
  id,
}: {
  item: any;
  id: string;
}) {
  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>Product #{id}</Title>
        <Space>
          <Button type="primary" href={`/products/${id}/edit`} icon={<EditOutlined />}>
            Edit
          </Button>
          <DeleteButton id={item.id} />
        </Space>
      </div>

      <Descriptions bordered column={1}>
        <Descriptions.Item label="ID">{item.id}</Descriptions.Item>
        <Descriptions.Item label="Name">{String(item.name ?? "")}</Descriptions.Item>
        <Descriptions.Item label="Amount">{String(item.amount ?? "")}</Descriptions.Item>
      </Descriptions>

      <div style={{ marginTop: 24 }}>
        <Button type="link" href="/products" icon={<ArrowLeftOutlined />} style={{ paddingLeft: 0 }}>
          Back to Products
        </Button>
      </div>
    </div>
  );
}
