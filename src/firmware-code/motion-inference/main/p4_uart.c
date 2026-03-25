#include <stdio.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/uart.h"
#include "driver/gpio.h"
#include "esp_log.h"

static const char *TAG = "P4_UART";

// 根据原理图确认以下引脚编号
#define P4_UART_NUM        UART_NUM_1        // 使用UART1
#define P4_TX_PIN          GPIO_NUM_26       // 需根据原理图修改
#define P4_RX_PIN          GPIO_NUM_27       // 需根据原理图修改
#define BUF_SIZE           1024
#define BAUD_RATE          115200

void uart_init(void)
{
    uart_config_t uart_config = {
        .baud_rate = BAUD_RATE,
        .data_bits = UART_DATA_8_BITS,
        .parity = UART_PARITY_DISABLE,
        .stop_bits = UART_STOP_BITS_1,
        .flow_ctrl = UART_HW_FLOWCTRL_DISABLE,
        .source_clk = UART_SCLK_DEFAULT,
    };

    // 安装UART驱动
    uart_driver_install(P4_UART_NUM, BUF_SIZE, BUF_SIZE, 0, NULL, 0);
    uart_param_config(P4_UART_NUM, &uart_config);
    uart_set_pin(P4_UART_NUM, P4_TX_PIN, P4_RX_PIN, UART_PIN_NO_CHANGE, UART_PIN_NO_CHANGE);
    
    ESP_LOGI(TAG, "UART initialized on TX=%d, RX=%d", P4_TX_PIN, P4_RX_PIN);
}

void uart_send_data(const char* data)
{
    uart_write_bytes(P4_UART_NUM, data, strlen(data));
    ESP_LOGI(TAG, "Sent: %s", data);
}

void uart_receive_task(void *pvParameter)
{
    uint8_t* data = (uint8_t*) malloc(BUF_SIZE);
    while (1) {
        int len = uart_read_bytes(P4_UART_NUM, data, BUF_SIZE, pdMS_TO_TICKS(100));
        if (len > 0) {
            data[len] = '\0';
            ESP_LOGI(TAG, "Received: %s", (char*)data);
            
            // 回声响应
            uart_write_bytes(P4_UART_NUM, (const char*)data, len);
            ESP_LOGI(TAG, "Echo sent back");
        }
        vTaskDelay(pdMS_TO_TICKS(10));
    }
    free(data);
    vTaskDelete(NULL);
}

void app_main(void)
{
    uart_init();
    
    // 创建接收任务
    xTaskCreate(uart_receive_task, "uart_rx_task", 4096, NULL, 10, NULL);
    
    // 主循环发送测试消息
    int count = 0;
    char send_buf[64];
    while (1) {
        sprintf(send_buf, "Hello from P4, count: %d\n", count++);
        uart_send_data(send_buf);
        vTaskDelay(pdMS_TO_TICKS(5000));
    }
}