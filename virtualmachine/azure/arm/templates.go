package arm

// See https://github.com/Azure/azure-quickstart-templates for a extensive list of templates.
const defaultTemplate = `{
   "$schema": "http://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
   "contentVersion": "",
   "parameters": {  },
   "variables": {  },
   "resources": [  ],
   "outputs": {  }
}`

const Linux2 = `{
  "$schema": "http://schema.management.azure.com/schemas/2014-04-01-preview/deploymentTemplate.json",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "adminUsername": {
      "type": "string"
    },
    "adminPassword": {
      "type": "string"
    },
    "dnsNameForPublicIP": {
      "type": "string"
    },
    "imagePublisher": {
      "type": "string"
    },
    "imageOffer": {
      "type": "string"
    },
    "imageSku": {
      "type": "string"
    },
    "osFile": {
      "type": "string"
    },
    "sshAuthorizedKey": {
      "type": "string"
    },
    "storageAccountName": {
      "type": "string"
    },
    "vmSize": {
      "type": "string"
    },
    "vmName": {
      "type": "string"
    }
  },
  "variables": {
    "addressPrefix": "10.0.0.0/16",
    "apiVersion": "2015-06-15",
    "location": "[resourceGroup().location]",
    "nicName": "apceraNic",
    "publicIPAddressName": "apceraPublicIP",
    "publicIPAddressType": "Dynamic",
    "sshKeyPath": "[concat('/home/',parameters('adminUsername'),'/.ssh/authorized_keys')]",
    "subnetName": "apceraSubnet",
    "subnetAddressPrefix": "10.0.0.0/24",
    "subnetRef": "[concat(variables('vnetID'),'/subnets/',variables('subnetName'))]",
    "virtualNetworkName": "apceraNetwork",
    "vmStorageAccountContainerName": "images",
    "vnetID": "[resourceId('Microsoft.Network/virtualNetworks', variables('virtualNetworkName'))]"
  },
  "resources": [
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/publicIPAddresses",
      "name": "[variables('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('dnsNameForPublicIP')]"
        }
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/virtualNetworks",
      "name": "[variables('virtualNetworkName')]",
      "location": "[variables('location')]",
      "properties": {
        "addressSpace": {
          "addressPrefixes": [
            "[variables('addressPrefix')]"
          ]
        },
        "subnets": [
          {
            "name": "[variables('subnetName')]",
            "properties": {
              "addressPrefix": "[variables('subnetAddressPrefix')]"
            }
          }
        ]
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/networkInterfaces",
      "name": "[variables('nicName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', variables('publicIPAddressName'))]",
        "[concat('Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'))]"
      ],
      "properties": {
        "ipConfigurations": [
          {
            "name": "ipconfig",
            "properties": {
              "privateIPAllocationMethod": "Dynamic",
              "publicIPAddress": {
                "id": "[resourceId('Microsoft.Network/publicIPAddresses', variables('publicIPAddressName'))]"
              },
              "subnet": {
                "id": "[variables('subnetRef')]"
              }
            }
          }
        ]
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Compute/virtualMachines",
      "name": "[parameters('vmName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/networkInterfaces/', variables('nicName'))]"
      ],
      "properties": {
        "hardwareProfile": {
          "vmSize": "[parameters('vmSize')]"
        },
        "osProfile": {
          "computerName": "[parameters('vmName')]",
          "adminUsername": "[parameters('adminUsername')]",
          "adminPassword": "[parameters('adminPassword')]",
          "linuxConfiguration": {
            "disablePasswordAuthentication": "false",
            "ssh": {
              "publicKeys": [
                {
                  "path": "[variables('sshKeyPath')]",
                  "keyData": "[parameters('sshAuthorizedKey')]"
                }
              ]
            }
          }
        },
        "storageProfile": {
          "imageReference": {
            "publisher": "[parameters('imagePublisher')]",
            "offer": "[parameters('imageOffer')]",
            "sku": "[parameters('imageSku')]",
            "version": "latest"
          },
          "osDisk": {
            "name": "osdisk",
            "vhd": {
              "uri": "[concat('http://',parameters('storageAccountName'),'.blob.core.windows.net/',variables('vmStorageAccountContainerName'),'/', parameters('osDiskName'),'.vhd')]"
            },
            "caching": "ReadWrite",
            "createOption": "FromImage"
          }
        },
        "networkProfile": {
          "networkInterfaces": [
            {
              "id": "[resourceId('Microsoft.Network/networkInterfaces', variables('nicName'))]"
            }
          ]
        },
        "diagnosticsProfile": {
          "bootDiagnostics": {
             "enabled": "false"
          }
        }
      }
    }
  ]
}`

const Linux = `{
  "$schema": "http://schema.management.azure.com/schemas/2014-04-01-preview/deploymentTemplate.json",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "adminUsername": {
      "type": "string"
    },
    "adminPassword": {
      "type": "string"
    },
    "imagePublisher": {
      "type": "string"
    },
    "imageOffer": {
      "type": "string"
    },
    "imageSku": {
      "type": "string"
    },
    "nicName": {
      "type": "string"
    },    
    "osFileName": {
      "type": "string"
    },
    "publicIPName": {
      "type": "string"
    },
    "sshAuthorizedKey": {
      "type": "string"
    },
    "storageAccountName": {
      "type": "string"
    },
    "subnetName": {
      "type": "string"
    },
    "virtualNetworkName": {
      "type": "string"
    },
    "vmSize": {
      "type": "string"
    },
    "vmName": {
      "type": "string"
    }
  },
  "variables": {
    "apiVersion": "2015-06-15",
    "location": "[resourceGroup().location]",
    "publicIPAddressType": "Dynamic",
    "vmStorageAccountContainerName": "images",
    "subnetRef": "[concat(variables('vnetID'),'/subnets/',parameters('subnetName'))]",
    "sshKeyPath": "[concat('/home/',parameters('adminUsername'),'/.ssh/authorized_keys')]",
    "vnetID": "[resourceId('Microsoft.Network/virtualNetworks', parameters('virtualNetworkName'))]"
  },
  "resources": [
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/publicIPAddresses",
      "name": "[parameters('publicIPName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('publicIPName')]"
        }
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/networkInterfaces",
      "name": "[parameters('nicName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', parameters('publicIPName'))]"
      ],
      "properties": {
        "ipConfigurations": [
          {
            "name": "ipconfig",
            "properties": {
              "privateIPAllocationMethod": "Dynamic",
              "publicIPAddress": {
                "id": "[resourceId('Microsoft.Network/publicIPAddresses', parameters('publicIPName'))]"
              },
              "subnet": {
                "id": "[variables('subnetRef')]"
              }
            }
          }
        ]
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Compute/virtualMachines",
      "name": "[parameters('vmName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/networkInterfaces/', parameters('nicName'))]"
      ],
      "properties": {
        "hardwareProfile": {
          "vmSize": "[parameters('vmSize')]"
        },
        "osProfile": {
          "computerName": "[parameters('vmName')]",
          "adminUsername": "[parameters('adminUsername')]",
          "adminPassword": "[parameters('adminPassword')]",
          "linuxConfiguration": {
            "disablePasswordAuthentication": "false"
          }
        },
        "storageProfile": {
          "imageReference": {
            "publisher": "[parameters('imagePublisher')]",
            "offer": "[parameters('imageOffer')]",
            "sku": "[parameters('imageSku')]",
            "version": "latest"
          },
          "osDisk": {
            "name": "osdisk",
            "vhd": {
              "uri": "[concat('http://',parameters('storageAccountName'),'.blob.core.windows.net/',variables('vmStorageAccountContainerName'),'/', parameters('osFileName'))]"
            },
            "caching": "ReadWrite",
            "createOption": "FromImage"
          }
        },
        "networkProfile": {
          "networkInterfaces": [
            {
              "id": "[resourceId('Microsoft.Network/networkInterfaces', parameters('nicName'))]"
            }
          ]
        },
        "diagnosticsProfile": {
          "bootDiagnostics": {
             "enabled": "false"
          }
        }
      }
    }
  ]
}`
