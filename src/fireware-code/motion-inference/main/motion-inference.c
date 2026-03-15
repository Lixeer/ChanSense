#include <stdio.h>
#include "esp_err.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_log_level.h"
#include "esp_netif.h"
#include "esp_netif_types.h"
#include "esp_wifi_default.h"
#include "esp_wifi_types_generic.h"
#include "freertos/FreeRTOS.h"
#include "freertos/idf_additions.h"
#include "freertos/projdefs.h"
#include "esp_wifi.h"
#include "nvs_flash.h"

#define WIFI_SSID ""
#define WIFI_PASSWORD ""
#define TAG "esp32p4"

EventGroupHandle_t wifi_event_group;

static uint8_t WIFI_AUTO_RECONNECT_ENABLED = 0;


static void wifi_event_handler_STA(void *arg, esp_event_base_t event_base,int32_t event_id, void *event_data) {
    // 处理WIFI事件
    if (event_base == WIFI_EVENT) {
        switch (event_id) {
            case WIFI_EVENT_STA_START:
                // 记录WIFI驱动就绪日志
                ESP_LOGI(TAG, "WiFi驱动就绪");
                break;
            case WIFI_EVENT_STA_CONNECTED:
                // 记录成功连接到热点日志
                ESP_LOGI(TAG, "已连接到热点");
                break;
            case WIFI_EVENT_STA_DISCONNECTED:
                // 记录热点断开日志并启用自动重连
                ESP_LOGW(TAG, "热点断开, 尝试重连...");
                WIFI_AUTO_RECONNECT_ENABLED = 1;
                break;
        }
    // 处理IP事件
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        // 获取IP事件数据
        ip_event_got_ip_t *event = (ip_event_got_ip_t*) event_data;
        // 记录成功获取IP日志
        ESP_LOGI(TAG, "成功获取IP:" IPSTR, IP2STR(&event->ip_info.ip));
    }
}




void wifi_init_sta(void)
{
    wifi_event_group = xEventGroupCreate();
    
    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());
    esp_netif_create_default_wifi_sta();
    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();


    ESP_ERROR_CHECK(esp_wifi_init(&cfg));
    ESP_LOGI("I", "wifi init successful");


    esp_event_handler_instance_t wifi_event_handle, ip_event_handle;
    esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID,&wifi_event_handler_STA, NULL, &wifi_event_handle);
    esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP,&wifi_event_handler_STA, NULL, &ip_event_handle);
    
    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASSWORD,
        },
    };

    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(ESP_IF_WIFI_STA, &wifi_config));
    if(esp_wifi_start() == ESP_OK){
        ESP_LOGI("I", "wifi start successful");
    }
    esp_wifi_connect();






}



void app_main(void)
{   

    /*
    while (1) {
        vTaskDelay(pdMS_TO_TICKS(1000));
        printf("Hello World!\n");
        
        
    }
    */
    printf("正在启动");
    esp_err_t ret = nvs_flash_init();

    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }


    wifi_init_sta();
    while (1) {
        vTaskDelay(pdMS_TO_TICKS(1000));
    
    }
}
