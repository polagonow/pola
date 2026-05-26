"use client";

import { Table, Button, Space, Typography } from "antd";
import { PlusOutlined, EyeOutlined, EditOutlined } from "@ant-design/icons";
import DeleteButton from "@/components/products/delete-button";

const { Title } = Typography;

export default function ProductsListView({
  items,
}: {
  items: any[];
}) {
  const columns = [
    { title: "ID", dataIndex: "id", key: "id" },
    {
      title: "Name",
      dataIndex: "name",
      key: "name",
      render: (value: unknown) => String(value ?? ""),
      },
    {
      title: "Amount",
      dataIndex: "amount",
      key: "amount",
      render: (value: unknown) => String(value ?? ""),
      },
    {
      title: "Actions",
      key: "actions",
      align: "right" as const,
      render: (_: unknown, record: { id: number }) => (
        <Space>
          <Button type="link" size="small" href={`/products/${record.id}`} icon={<EyeOutlined />}>
            View
          </Button>
          <Button type="link" size="small" href={`/products/${record.id}/edit`} icon={<EditOutlined />}>
            Edit
          </Button>
          <DeleteButton id={record.id} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>Products</Title>
        <Button type="primary" href="/products/create" icon={<PlusOutlined />}>
          New Product
        </Button>
      </div>

      <Table
        dataSource={items}
        columns={columns}
        rowKey="id"
        pagination={false}
        locale={{ emptyText: "No products found." }}
      />
    </div>
  );
}
